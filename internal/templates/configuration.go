package templates

import (
	"bytes"
	frpv1 "frp-operator/api/v1"
	"frp-operator/internal/constants"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

type TemplateData struct {
	ExitServer   *frpv1.ExitServer
	Token        *string
	AdminAPIPort int
	Tunnels      *[]frpv1.Tunnel
}

const CONFIGURATION = `
{{- print -}}
serverAddr = {{ quote .ExitServer.Spec.Host }}
serverPort = {{ .ExitServer.Spec.Port }}

{{- with .Token }}
auth.method = "token"
auth.token = {{ quote . }}
{{- end }}

webServer.addr = "0.0.0.0"
webServer.port = {{ .AdminAPIPort }}

{{- range $tunnel := .Tunnels }}

[[proxies]]
name = {{ quote $tunnel.Name }}
{{- if $tunnel.Spec.TCP }}
type = "tcp"
localIP = {{ serviceAddress $tunnel.Spec.TCP.ServiceRef }}
localPort = {{ $tunnel.Spec.TCP.LocalPort }}
remotePort = {{ $tunnel.Spec.TCP.RemotePort }}
{{- else if $tunnel.Spec.HTTP }}
type = "http"
{{- with $tunnel.Spec.HTTP.CustomDomains }}
customDomains = {{ stringArray $tunnel.Spec.HTTP.CustomDomains }}
{{- end }}
{{- with $tunnel.Spec.HTTP.Subdomain }}
subdomain = {{ quote . }}
{{- end }}
{{- with $tunnel.Spec.HTTP.Locations }}
locations = {{ stringArray . }}
{{- end }}
{{- with $tunnel.Spec.HTTP.HTTPUser }}
httpUser = {{ quote . }}
{{- end }}
{{- with $tunnel.Spec.HTTP.HTTPPassword }}
httpPassword = {{ quote . }}
{{- end }}
{{- with $tunnel.Spec.HTTP.HostHeaderRewrite }}
hostHeaderRewrite = {{ quote . }}
{{- end }}
{{- with $tunnel.Spec.HTTP.RouteByHTTPUser }}
routeByHTTPUser = {{ quote . }}
{{- end }}
{{- range $key, $value := $tunnel.Spec.HTTP.RequestHeaders }}
requestHeaders.set.{{ tomlKey $key }} = {{ quote $value }}
{{- end }}
{{- range $key, $value := $tunnel.Spec.HTTP.ResponseHeaders }}
responseHeaders.set.{{ tomlKey $key }} = {{ quote $value }}
{{- end }}
{{- if $tunnel.Spec.HTTP.Plugin }}

[proxies.plugin]
type = {{ quote $tunnel.Spec.HTTP.Plugin.Type }}
localAddr = {{ localAddress $tunnel.Spec.HTTP.Plugin.ServiceRef $tunnel.Spec.HTTP.Plugin.LocalPort }}
{{- with $tunnel.Spec.HTTP.Plugin.HostHeaderRewrite }}
hostHeaderRewrite = {{ quote . }}
{{- end }}
{{- range $key, $value := $tunnel.Spec.HTTP.Plugin.RequestHeaders }}
requestHeaders.set.{{ tomlKey $key }} = {{ quote $value }}
{{- end }}
{{- else }}
localIP = {{ serviceAddress $tunnel.Spec.HTTP.ServiceRef }}
localPort = {{ $tunnel.Spec.HTTP.LocalPort }}
{{- end }}
{{- end }}
{{- with $tunnel.Spec.Transport }}
{{- with $tunnel.Spec.Transport.UseEncryption }}
transport.useEncryption = {{ . }}
{{- end }}
{{- with $tunnel.Spec.Transport.UseCompression }}
transport.useCompression = {{ . }}
{{- end }}
{{- with $tunnel.Spec.Transport.ProxyProtocol }}
transport.proxyProtocolVersion = "{{ . }}"
{{- end }}
{{- with $tunnel.Spec.Transport.BandwidthLimit }}
transport.bandwidthLimit = "{{ . }}"
{{- with $tunnel.Spec.Transport.BandwidthLimitMode }}
transport.bandwidthLimitMode = "{{ . }}"
{{- else }}
transport.bandwidthLimitMode = "server"
{{- end }}
{{- end }}
{{- end -}}
{{- end -}}
`

func CreateConfiguration(exitServer *frpv1.ExitServer, token string, tunnels []frpv1.Tunnel) (string, error) {
	templateEngine, err := template.New("configuration-template").Funcs(template.FuncMap{
		"localAddress":   localAddress,
		"quote":          strconv.Quote,
		"serviceAddress": serviceAddress,
		"stringArray":    stringArray,
		"tomlKey":        strconv.Quote,
	}).Parse(CONFIGURATION)
	if err != nil {
		return "", err
	}

	sort.Slice(tunnels[:], func(i, j int) bool {
		return tunnels[i].Name < tunnels[j].Name
	})

	var buffer bytes.Buffer
	err = templateEngine.Execute(&buffer, TemplateData{
		ExitServer:   exitServer,
		Token:        &token,
		AdminAPIPort: constants.AdminAPIPort,
		Tunnels:      &tunnels,
	})
	if err != nil {
		return "", err
	}

	return buffer.String(), nil
}

func serviceAddress(serviceRef frpv1.ServiceRef) string {
	return strconv.Quote(rawServiceAddress(serviceRef))
}

func rawServiceAddress(serviceRef frpv1.ServiceRef) string {
	address := serviceRef.Name
	if serviceRef.Namespace != nil {
		address += "." + *serviceRef.Namespace + ".svc"
	}

	return address
}

func localAddress(serviceRef frpv1.ServiceRef, port int) string {
	return strconv.Quote(rawServiceAddress(serviceRef) + ":" + strconv.Itoa(port))
}

func stringArray(values []string) string {
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = strconv.Quote(value)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
