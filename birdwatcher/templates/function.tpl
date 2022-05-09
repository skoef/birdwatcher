function {{.FunctionName}}()
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
