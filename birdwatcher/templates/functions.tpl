# DO NOT EDIT MANUALLY
{{- range .Collections }}
function {{.FunctionName}}(){{- if not $.CompatBird213 }} -> bool{{- end }}
{
{{- with .Prefixes}}
	return net ~ [
{{- range prefixPad . }}
		{{.}}
{{- end }}
	];
{{- else }}
	return false;
{{- end }}
}
{{- end }}
