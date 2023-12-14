function {{.FunctionName}}() -> bool
{
{{- with .Prefixes}}
	return net ~ [
{{- range prefpad . }}
		{{.}}
{{- end }}
	];
{{- else }}
	return false;
{{- end }}
}
