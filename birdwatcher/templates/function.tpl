function {{.FunctionName}}(){{- if not .NoReturnType }} -> bool {{- end }}
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
