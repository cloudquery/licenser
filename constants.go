package main

const template = `---
hub-title: Licenses
---

The following tools / packages are used in this plugin:

| Name | License |
|------|---------|
{{- range . }}
{{- if ne .LicenseName "Unknown" }}
| {{ .Name }} | {{ .LicenseName }} |
{{- end }}
{{- end }}`
