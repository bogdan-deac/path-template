package path_template

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestMatchSuccess(t *testing.T) {
	validPathTemplates := []string{
		"/", "/a", "/abc", "/a/b", "/ab/c/d/e", "/a/",
		"/*", "/a/*", "/*/a", "/a/*/b", "/*/*", "/*/a/*", "/*/",
		"/**", "/a/**", "/*/a/**", "/a/*/b/**", "/*/**", "/*/*/**", "/**/a", "/**/a/b",
		"/**.m3u8", "/**.mpd", "/*_suf", "/{path=**}.m3u8", "/{foo}/**.ts",
		"/media/*.m4s", "/media/{contentId=*}/**", "/media/*", "/api/*/*/**",
		"/api/*/v1/**", "/api/*/v1/*", "/{version=api/*}/*", "/api/*/*/",
		"/api/*/1234/", "/api/*/{resource=*}/{method=*}",
		"/api/*/{resource=*}/{method=**}", "/v1/**", "/media/{country}/{lang=*}/**",
		"/{foo}/{bar}/{fo}/{fum}/*", "/{foo=*}/{bar=*}/{fo=*}/{fum=*}/*",
		"/media/{id=*}/*", "/media/{contentId=**}",
		"/api/{version}/projects/{project}/locations/{location}/{resource}/",
		"/api/{version=*}/{url=**}", "/api/{VERSION}/{version}/{verSION}",
		"/api/1234/abcd", "/media/abcd/%10%20%30/{v1=*/%10%20}_suffix",
		"/*aA0-._~%20!$&'()+,;:@=", "/**aA0-._~%20!$&'()+,;:@=",
		"/{foo}aA0-._~%20!$&'()+,;:@=", "/{foo=bar}aA0-._~%20!$&'()+,;:@=",
		"/{foo=*/bar}aA0-._~%20!$&'()+,;:@=", "/{foo=**}aA0-._~%20!$&'()+,;:@=",
		"/{foo=*/**}aA0-._~%20!$&'()+,;:@=",
	}

	for _, path := range validPathTemplates {
		_, err := ValidatePathTemplate(path)
		assert.NilError(t, err)
	}
}

func TestMatchFailure(t *testing.T) {
	tt := []struct {
		path string
		err  string
	}{
		{
			path: "/a//b",
			err:  "Empty segment not allowed in path template: a//b",
		},
		{
			path: "/**/*",
			err:  "Cannot have path glob (*) after text glob (**)",
		},
		{
			path: "/{a-b}",
			err:  "Variable name must start with a letter and contain only alphanumeric characters and underscores: a-b",
		},
		{
			path: "/**/{a=*}",
			err:  "Cannot have variable after text glob (**): {a=*}",
		},
		{
			path: "/**/{ext=**}",
			err:  "Cannot have variable after text glob (**): {ext=**}",
		},
		{
			path: "/api/v*/1234",
			err:  "Prefixes not allowed before operators: v*",
		},
		{
			path: "/api/v*.0",
			err:  "Prefixes not allowed before operators: v*.0",
		},
		{
			path: "/api/{version=v*}/1234",
			err:  "Prefixes or suffixes not allowed with variable pattern operators: v*",
		},
		{
			path: "/api/{version=v1*}/1234",
			err:  "Prefixes or suffixes not allowed with variable pattern operators: v1*",
		},
		{
			path: "/api/{version=*beta}/1234",
			err:  "Prefixes or suffixes not allowed with variable pattern operators: *beta",
		},
		{
			path: "/api/{version=*beta}/*1234",
			err:  "Prefixes or suffixes not allowed with variable pattern operators: *beta",
		},
		{
			path: "/media/eff456/ll-sd-out.{ext}",
			err:  "Prefixes not allowed before operators: ll-sd-out.{ext}",
		},
		{
			path: "/media/eff456/ll-sd-out.{ext=*}",
			err:  "Prefixes not allowed before operators: ll-sd-out.{ext=*}",
		},
		{
			path: "/media/eff456/ll-sd-out.**",
			err:  "Prefixes not allowed before operators: ll-sd-out.**",
		},
		{
			path: "/media/{country=**}/{lang=*}/**",
			err:  "Cannot have variable after text glob (**): {lang=*}",
		},
		{
			path: "/media/**/**/**",
			err:  "Cannot have text glob (**) after text glob (**)",
		},
		{
			path: "/link/{id=*}/asset*",
			err:  "Prefixes not allowed before operators: asset*",
		},
		{
			path: "/link/{id=*}/{asset=asset*}",
			err:  "Prefixes or suffixes not allowed with variable pattern operators: asset*",
		},
		{
			path: "/link/{id=*}/{asset=asset*-v1}",
			err:  "Prefixes or suffixes not allowed with variable pattern operators: asset*-v1",
		},
		{
			path: "/media/{id=/*}/*",
			err:  "Variable pattern cannot start or end with a slash: /*",
		},
		{
			path: "/media/{contentId=/**}",
			err:  "Variable pattern cannot start or end with a slash: /**",
		},
		{
			path: "/media/{contentId=**/}",
			err:  "Variable pattern cannot start or end with a slash: **/",
		},
		{
			path: "/media/{contentId=/**/}",
			err:  "Variable pattern cannot start or end with a slash: /**/",
		},
		{
			path: "/api/{version}/{version}",
			err:  "Variable name is duplicated: version",
		},
		{
			path: "/api/{version}/{version=**}",
			err:  "Variable name is duplicated: version",
		},
		{
			path: "/api/{version.major}/{version.minor}",
			err:  "Variable name must start with a letter and contain only alphanumeric characters and underscores: version.major",
		},
		{
			path: "/media/***",
			err:  "Invalid segment in path template: ***",
		},
		{
			path: "/media/*{*}*",
			err:  "Invalid segment in path template: *{*}*",
		},
		{
			path: "/media/{*}/",
			err:  "Variable name must start with a letter and contain only alphanumeric characters and underscores: *",
		},
		{
			path: "/media/*/index?a=2",
			err:  "Invalid segment in path template: index?a=2",
		},
		{
			path: "media",
			err:  "PathTemplate must start with a /: media",
		},
		{
			path: "{media}",
			err:  "PathTemplate must start with a /: {media}",
		},
		{
			path: "/\x01\x02\x03\x04\x05\x06",
			err:  "PathTemplate contains non-representable characters: /\x01\x02\x03\x04\x05\x06",
		},
		{
			path: "/*(/**",
			err:  "The suffixed operator must in be the final path component: /*(/**",
		},
		{
			path: "/**/{var}",
			err:  "Cannot have variable after text glob (**): {var}",
		},
		{
			path: "/{var1}/{var2}/{var3}/{var4}/{var5}/{var6}",
			err:  "Cannot have more than 5 variables: /{var1}/{var2}/{var3}/{var4}/{var5}/{var6}",
		},
		{
			path: "/{var1=*}/{var2=*}/{var3=*}/{var4=*}/{var5=*}/{var6=*}",
			err:  "Cannot have more than 5 variables: /{var1=*}/{var2=*}/{var3=*}/{var4=*}/{var5=*}/{var6=*}",
		},
		{
			path: "/{=*}",
			err:  "Variable name cannot be empty: /{=*}",
		},
		{
			path: "/{var12345678901234=*}",
			err:  "Variable name exceeds 16 characters: var12345678901234",
		},
		{
			path: "/{var=*/{var1}/x}",
			err:  "Nested brackets not allowed in path template: {var=*/{var1}/x}",
		},
		{
			path: "/{var=*/***/x}",
			err:  "Invalid variable pattern segment: ***",
		},
		{
			path: "/{api",
			err:  "Unmatched { not allowed in path template: {api",
		},
		{
			path: "/api}",
			err:  "Unmatched } not allowed in path template: api}",
		},
		{
			path: "/{{api}}",
			err:  "Nested brackets not allowed in path template: {{api}}",
		},
		{
			path: "/{2bOrNot2b}",
			err:  "Variable name must start with a letter and contain only alphanumeric characters and underscores: 2bOrNot2b",
		},
		{
			path: "/{nowIsTheWinterOfOurDiscontent}",
			err:  "Variable name exceeds 16 characters: nowIsTheWinterOfOurDiscontent",
		},
		{
			path: `/""`,
			err:  `Invalid segment in path template: ""`,
		},
		{
			path: "/{a/*}",
			err:  "Variable name must start with a letter and contain only alphanumeric characters and underscores: a/*",
		},
		{
			path: "/api/v1/invites{service-path=**}",
			err:  "Prefixes not allowed before operators: invites{service-path=**}",
		},
	}

	for _, tc := range tt {
		_, err := ValidatePathTemplate(tc.path)
		assert.Error(t, err, tc.err)
	}

}

func TestPathTemplateRewriteSuccess(t *testing.T) {
	validRewrites := []string{
		"/{var1}", "/{var1}{var2}", "/{var1}-{var2}",
		"/abc/{var1}/def", "/{var1}/abd/{var2}",
		"/abc-def-{var1}/a/{var1}",
	}
	for _, rewrite := range validRewrites {
		_, err := validatePathTemplateRewriteSyntax(rewrite)
		assert.NilError(t, err)
	}
}

func TestPathTemplateRewriteFailure(t *testing.T) {
	tt := []struct {
		rewrite string
		err     string
	}{
		{
			rewrite: "/{var1",
			err:     "Unmatched { not allowed in path template rewrite: /{var1",
		},
		{
			rewrite: "/{{var1}}",
			err:     "Nested brackets in not allowed in path template rewrite: /{{var1}}",
		},
		{
			rewrite: "{var1}",
			err:     "Replace path template must start with a /: {var1}",
		},
		{
			rewrite: "/}va1{",
			err:     "Unmatched } not allowed in path template rewrite: /}va1{",
		},
		{
			rewrite: "var1}",
			err:     "Replace path template must start with a /: var1}",
		},
		{
			rewrite: "/{var1}?abc=123",
			err:     "Invalid character found in path template rewrite: /{var1}?abc=123",
		},
		{
			rewrite: "",
			err:     "Replace path template must start with a /: ",
		},
		{
			rewrite: "/{var1/var2}",
			err:     "Variable name must start with a letter and contain only alphanumeric characters and underscores: var1/var2",
		},
		{
			rewrite: "/{}",
			err:     "Empty variable not allowed in path template rewrite: /{}",
		},
		{
			rewrite: "/a//b",
			err:     "Empty segment not allowed in path template rewrite: /a//b",
		},
	}
	for _, tc := range tt {
		_, err := validatePathTemplateRewriteSyntax(tc.rewrite)
		assert.Error(t, err, tc.err)
	}
}

func TestPathTemplateMatchRewriteSuccess(t *testing.T) {
	tt := []struct {
		match   string
		rewrite string
	}{
		{
			match:   "/{var1}",
			rewrite: "/{var1}",
		},
		{
			match:   "/api/users/{id}/{path=**}",
			rewrite: "/users/{id}/{path}",
		},
		{
			match:   "/videos/*/{id}/{format}/{rendition}/{segment=**}.ts",
			rewrite: "/{id}/{format}/{rendition}/{segment}.ts",
		},
		{
			match:   "/region/{region}/bucket/{name}/{method=**}",
			rewrite: "/{region}/bucket-{name}/{method}",
		},
		{
			match:   "/region/{region}/bucket/{name}/{method=**}",
			rewrite: "/{region}{name}/{method}",
		},
		{
			match:   "/{a}",
			rewrite: "/{a}/{a}-a/a-{a}-{a}",
		},
		{
			match:   "/videos/*/{id}/{format}/{rendition}/{segment=**}.ts",
			rewrite: "/{segment}",
		},
	}
	for _, tc := range tt {
		variables, err := ValidatePathTemplate(tc.match)
		assert.NilError(t, err)
		err = ValidatePathTemplateRewrite(tc.rewrite, variables)
		assert.NilError(t, err)
	}
}
func TestPathTemplateMatchRewriteFailure(t *testing.T) {
	tt := []struct {
		match   string
		rewrite string
		err     string
	}{
		{
			match:   "/{var1}",
			rewrite: "/{var2}",
			err:     "Variable var2 in path template rewrite is not present in the path template: /{var2}",
		},
		{
			match:   "/api/users/{id}/{path=**}",
			rewrite: "/users/{id}/{path}/{extra}",
			err:     "Variable extra in path template rewrite is not present in the path template: /users/{id}/{path}/{extra}",
		},
	}
	for _, tc := range tt {
		variableNames, err := ValidatePathTemplate(tc.match)
		assert.NilError(t, err)
		err = ValidatePathTemplateRewrite(tc.rewrite, variableNames)
		assert.Error(t, err, tc.err)
	}
}
