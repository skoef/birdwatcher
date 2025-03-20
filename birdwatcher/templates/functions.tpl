# DO NOT EDIT MANUALLY
{{- range .Collections }}
function {{.FunctionName}}(){{- if not $.CompatBird213 }} -> bool{{- end }}
{
{{- if .EnablePrefixFilter }}
{{- with .Prefixes}}
	return net ~ [
{{- range prefixPad . }}
		{{.}}
{{- end }}{{/* end of range prefixPad */}}
	];
{{- else }}
	return false;
{{- end }}
{{- else }}{{/* else of if .enablePrefixFilter */}}
{{- with .Prefixes}}
	return true;
{{- else }}
	return false;
{{- end }}

{{- end }}{{/* end of if .enablePrefixFilter */}}
}
{{- end }}{{/* end of range .Collections */}}
