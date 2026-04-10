{{- .Title | replaceRE "\n" " " | printf "# %s" }}
{{ .RawContent | replaceRE `(?s)\{\{[<%].*` "" | strings.TrimSpace }}
