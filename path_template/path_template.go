package path_template

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

const (
	defaultEnvoyMaxNameLength = 16
	defaultEnvoyMinNameLength = 1
	// at most 5 variables - {foo} or {foo=bar}
	defaultEnvoyMaxVariablePerPath = 5

	textGlob = "*"
	pathGlob = "**"

	// valid pchar from https://datatracker.ietf.org/doc/html/rfc3986#appendix-A
	validLiteralSymbolsReS = "a-zA-Z0-9-._~" + // unreserved
		"%" + // pct-encoded
		"!$&'()+,;" + // sub-delims excluding * and =
		":@" +
		"=" // user included = allowed
)

var (

	//Regex to match a valid literal
	validLiteralRe = regexp.MustCompile("^[" + validLiteralSymbolsReS + "]+$")

	// graphically printable ascii characters - per GNU docs:
	// Graphical characters: ‘[:alnum:]’ and ‘[:punct:]’.
	rePrintable = regexp.MustCompile("^[[:graph:]]*$")

	// the range of possibilities for a variable name
	reVariableName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

	// reSuffixedSegment is used to match the suffix of a segment: {foo=**}-suffix
	reSuffixedSegment = regexp.MustCompile(`^(\*|\*\*|{.*})[` + validLiteralSymbolsReS + `]+$`)

	// rePrefixedOperator matches operators that have a prefix and may have a suffix
	// It is used for clearer error messages. Non capturing group - we don't need the operator
	rePrefixedOperator = regexp.MustCompile(`^[` + validLiteralSymbolsReS + `]+(?:\*|\*\*|{.*}).*$`)

	// rePrefixedSuffixedVariablePatternSegment matches path variable path segments
	// with prefixes and/or suffixes, which are not allowed. It is used for clearer error messages
	// Non-capturing group - we don't need the operator
	rePrefixedSuffixedVariablePatternSegment = regexp.MustCompile(
		`^[` + validLiteralSymbolsReS + `]*(?:\*|\*\*)[` + validLiteralSymbolsReS + `]*$`,
	)

	// a valid rewrite literal also includes the / character.
	// Slashes don't have special relevance with the exception of duplicate consecutive ones
	reValidTemplateRewriteLiteral = regexp.MustCompile(`^[` + validLiteralSymbolsReS + `/]*$`)
)

// Definitions
// path glob := * (matches a single path segment)
// text glob := ** (matches zero or more path segments)
// variable := {varName} a capturing version of a path glob
//          OR {varName=pattern} a capturing version for the specified pattern
// wildcard operator := path glob OR text glob OR variable

// implementation based on the original envoy code
// https://github.com/envoyproxy/envoy/blob/7edad229843b507dc10faeb44d2867a5da25b631/source/extensions/path/uri_template_lib/uri_template_internal.cc
// it validates the following
// syntax of path template
// number of variables (at most 5)
// length of variable names (at most 16)
// uniqueness of variable names
// syntax of variable patterns
func ValidatePathTemplate(path string) ([]string, error) {
	if !rePrintable.MatchString(path) {
		return nil, fmt.Errorf("PathTemplate contains non-representable characters: %s", path)
	}

	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("PathTemplate must start with a /: %s", path)
	}

	// at this point, valid path segments
	segments, err := parsePathTemplate(path)
	if err != nil {
		return nil, err
	}

	// PathTemplates may contain path globs, text globs and variables.
	// Variable patterns may contain path or text globs. If a wildcard operator is found anywhere
	// in the PathTemplate string, it must be the last (rightmost) wildcard operator.
	foundTextGlob := false

	// Suffixes are also allowed for wildcard operators (ie *-suffix or {name}-suffix).
	// If a suffixed wildcard operator is found, it must be the last (rightmost) wildcard operator in the PathTemplate string.
	foundSuffix := false

	variableNames := []string{}
	for _, segment := range segments {
		if foundSuffix {
			return nil, fmt.Errorf("The suffixed operator must in be the final path component: %s", path)
		}
		if reSuffixedSegment.MatchString(segment) {
			foundSuffix = true
			// extract the operator, that's what we need to validate - ie *, ** or {...}
			segment = reSuffixedSegment.FindStringSubmatch(segment)[1]
		}
		switch {
		// <..>/*/<..>
		case segment == textGlob:
			if foundTextGlob {
				return nil, fmt.Errorf("Cannot have path glob (*) after text glob (**)")
			}

		// <..>/**/<..>
		case segment == pathGlob:
			if foundTextGlob {
				return nil, fmt.Errorf("Cannot have text glob (**) after text glob (**)")
			}
			foundTextGlob = true

		// <..>/foo/<..>
		case validLiteralRe.MatchString(segment):
			continue

		// <..>/{<varSyntax>}/<..>
		case segment[0] == '{' && segment[len(segment)-1] == '}':
			if foundTextGlob {
				return nil, fmt.Errorf("Cannot have variable after text glob (**): %s", segment)
			}
			// <..>/{foo=bar}/<..>
			if strings.ContainsRune(segment, '=') {
				parts := strings.SplitN(segment, "=", 2)

				// {foo -> remove opening bracket
				name := parts[0][1:]

				if err := validateVariableName(name, path); err != nil {
					return nil, err
				}
				// two variables with the same name are not allowed - /{foo}/{foo=bar}
				if slices.Contains(variableNames, name) {
					return nil, fmt.Errorf("Variable name is duplicated: %s", name)
				}
				variableNames = append(variableNames, name)

				if len(variableNames) > defaultEnvoyMaxVariablePerPath {
					return nil, fmt.Errorf("Cannot have more than %d variables: %s", defaultEnvoyMaxVariablePerPath, path)
				}

				// bar} -> remove closing bracket
				pattern := parts[1][:len(parts[1])-1]

				// cannot have {foo=}
				if len(pattern) == 0 {
					return nil, fmt.Errorf("Variable pattern is empty for: %s", name)
				}
				if pattern[0] == '/' || pattern[len(pattern)-1] == '/' {
					return nil, fmt.Errorf("Variable pattern cannot start or end with a slash: %s", pattern)
				}
				for _, patternSegment := range strings.Split(pattern, "/") {
					switch {
					// {foo=<..>/*/<..>}
					case patternSegment == textGlob:
						if foundTextGlob {
							return nil, fmt.Errorf("Cannot have path glob (*) after text glob (**)")
						}

					// {foo=<..>/**/<..>}
					case patternSegment == pathGlob:
						if foundTextGlob {
							return nil, fmt.Errorf("Cannot have text glob (**) after text glob (**)")
						}
						foundTextGlob = true

					// {foo=<..>/bar/<..>}
					case !strings.ContainsRune(patternSegment, '*'):
						continue

					// {foo=<..>/prefix-**-suffix/<..>}
					case rePrefixedSuffixedVariablePatternSegment.MatchString(patternSegment):
						return nil, fmt.Errorf("Prefixes or suffixes not allowed with variable pattern operators: %s", patternSegment)

					default:
						return nil, fmt.Errorf("Invalid variable pattern segment: %s", patternSegment)
					}
				}
			} else {
				// <..>/{foo}/<..>

				// trim the curly braces
				name := segment[1 : len(segment)-1]

				if err := validateVariableName(name, path); err != nil {
					return nil, err
				}

				// two variables with the same name are not allowed - /{foo}/{foo=bar}
				if slices.Contains(variableNames, name) {
					return nil, fmt.Errorf("Variable name is duplicated: %s", name)
				}

				variableNames = append(variableNames, name)

				if len(variableNames) > defaultEnvoyMaxVariablePerPath {
					return nil, fmt.Errorf("Cannot have more than %d variables: %s", defaultEnvoyMaxVariablePerPath, path)
				}
			}
		// <..>/prefix{...}/<..> or <..>/prefix*/<..> or <..>/prefix**/<..>
		case rePrefixedOperator.MatchString(segment):
			return nil, fmt.Errorf("Prefixes not allowed before operators: %s", segment)
		default:
			return nil, fmt.Errorf("Invalid segment in path template: %s", segment)
		}
	}

	return variableNames, nil
}

// parsePathTemplate splits a path template into segments
// Example: /a/{foo}/b/{bar=*/**} -> [a, {foo}, b, {bar=*/**}]
func parsePathTemplate(path string) ([]string, error) {
	// remove leading slash
	path = path[1:]

	//split by slashes
	segments := []string{}

	// used for identifying the start of a new path segment - ie - what comes after /
	var segStart int
	var insideBrackets bool

	// parses everything that's between slashes which are not inside variables
	// {foo=*/x} is a valid path template so we can't just do a simple split
	for i, c := range path {
		switch c {
		case '/':
			// we can have patterns like {foo=a/*}
			if insideBrackets {
				continue
			}
			// this happens for cases like /a//b
			if segStart == i {
				return nil, fmt.Errorf("Empty segment not allowed in path template: %s", path)
			}
			segments = append(segments, path[segStart:i])
			segStart = i + 1
		case '{':
			if insideBrackets {
				return nil, fmt.Errorf("Nested brackets not allowed in path template: %s", path)
			}
			insideBrackets = true
		case '}':
			if !insideBrackets {
				return nil, fmt.Errorf("Unmatched } not allowed in path template: %s", path)
			}
			insideBrackets = false
		default:
		}
	}
	if insideBrackets {
		return nil, fmt.Errorf("Unmatched { not allowed in path template: %s", path)
	}

	// treat leftover segment if it exists -i.e /a/{b}/leftoverSegment
	if segStart != len(path) {
		segments = append(segments, path[segStart:])
	}

	return segments, nil
}

// Validates the correctness of a path template rewrite.
// Variable names not present in the match condition are not allowed
func ValidatePathTemplateRewrite(pathTemplateRewrite string, variableNames []string) error {
	rewriteVarNames, err := validatePathTemplateRewriteSyntax(pathTemplateRewrite)
	if err != nil {
		return err
	}

	for varName := range rewriteVarNames {
		if !slices.Contains(variableNames, varName) {
			return fmt.Errorf("Variable %s in path template rewrite is not present in the path template: %s", varName, pathTemplateRewrite)
		}
	}
	return nil

}

func validatePathTemplateRewriteSyntax(pathTemplateRewrite string) (map[string]bool, error) {
	// the rewrite field must start with a /
	if !strings.HasPrefix(pathTemplateRewrite, "/") {
		return nil, fmt.Errorf("Replace path template must start with a /: %s", pathTemplateRewrite)
	}

	insideBrackets := false
	rewriteVarNames := make(map[string]bool)
	var startIndex int
	for i, c := range pathTemplateRewrite {
		switch c {
		case '{':
			if insideBrackets {
				return nil, fmt.Errorf("Nested brackets in not allowed in path template rewrite: %s", pathTemplateRewrite)
			}
			insideBrackets = true
			if startIndex != i {
				literal := pathTemplateRewrite[startIndex:i]
				if !reValidTemplateRewriteLiteral.MatchString(literal) {
					return nil, fmt.Errorf("Invalid character in path template rewrite: %s", pathTemplateRewrite)
				}
			}
			startIndex = i + 1
		case '}':
			if !insideBrackets {
				return nil, fmt.Errorf("Unmatched } not allowed in path template rewrite: %s", pathTemplateRewrite)
			}
			insideBrackets = false

			if startIndex == i {
				return nil, fmt.Errorf("Empty variable not allowed in path template rewrite: %s", pathTemplateRewrite)
			}
			// take what's between the brackets - that's the name
			varName := pathTemplateRewrite[startIndex:i]

			if err := validateVariableName(varName, pathTemplateRewrite); err != nil {
				return nil, err
			}

			// we don't care if we have the same variable twice here
			// /{a}/{b}/{a} is a valid rewrite
			rewriteVarNames[varName] = true
			startIndex = i + 1
		case '/':
			if i < len(pathTemplateRewrite)-1 && pathTemplateRewrite[i+1] == '/' {
				return nil, fmt.Errorf("Empty segment not allowed in path template rewrite: %s", pathTemplateRewrite)
			}
		}
	}
	if insideBrackets {
		return nil, fmt.Errorf("Unmatched { not allowed in path template rewrite: %s", pathTemplateRewrite)
	}

	// treat leftover literal case  /a/{var1}abcd
	if startIndex != len(pathTemplateRewrite) {
		literal := pathTemplateRewrite[startIndex:]
		if !reValidTemplateRewriteLiteral.MatchString(literal) {
			return nil, fmt.Errorf("Invalid character found in path template rewrite: %s", pathTemplateRewrite)
		}
	}

	return rewriteVarNames, nil
}

func validateVariableName(name, fullString string) error {
	if len(name) < defaultEnvoyMinNameLength {
		return fmt.Errorf("Variable name cannot be empty: %s", fullString)
	}

	if !reVariableName.MatchString(name) {
		return fmt.Errorf("Variable name must start with a letter and contain only alphanumeric characters and underscores: %s", name)
	}
	if len(name) > 16 {
		return fmt.Errorf("Variable name exceeds 16 characters: %s", name)
	}
	return nil
}
