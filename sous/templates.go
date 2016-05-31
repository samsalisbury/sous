// This file was automatically generated based on the contents of *.tmpl
// If you need to update this file, change the contents of those files
// (or add new ones) and run 'go generate'

package sous 

const (
metadataDockerfileTmpl = "FROM {{.ImageID}}\nLABEL\n  {{- range $key, $value := .Labels}} \\\n  {{$key}}={{$value}}\n  {{- end -}}\n"

)
