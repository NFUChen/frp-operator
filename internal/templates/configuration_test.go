package templates

import (
	"strings"
	"testing"

	frpv1 "frp-operator/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ptr[T any](value T) *T {
	return &value
}

func exitServer() *frpv1.ExitServer {
	return &frpv1.ExitServer{
		Spec: frpv1.ExitServerSpec{
			Host: "frps.example.com",
			Port: 7000,
		},
	}
}

func TestCreateConfigurationRendersTCPProxy(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "tcp-tunnel"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			TCP: &frpv1.TCP{
				ServiceRef: frpv1.ServiceRef{Name: "my-svc", Namespace: ptr("my-ns")},
				LocalPort:  1234,
				RemotePort: 1234,
			},
			Transport: &frpv1.Transport{UseEncryption: true},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`[[proxies]]`,
		`name = "tcp-tunnel"`,
		`type = "tcp"`,
		`localIP = "my-svc.my-ns.svc"`,
		`localPort = 1234`,
		`remotePort = 1234`,
		`transport.useEncryption = true`,
	}, "\n")

	if !strings.Contains(configuration, expected) {
		t.Fatalf("expected configuration to contain:\n%s\ngot:\n%s", expected, configuration)
	}
}

func TestCreateConfigurationRendersHTTPProxyWithPlugin(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "guest-k8s-traefik"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			HTTP: &frpv1.HTTP{
				CustomDomains: []string{"*.example.com"},
				Plugin: &frpv1.Plugin{
					Type:       "http2http",
					ServiceRef: frpv1.ServiceRef{Name: "traefik", Namespace: ptr("traefik")},
					LocalPort:  80,
				},
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`[[proxies]]`,
		`name = "guest-k8s-traefik"`,
		`type = "http"`,
		`customDomains = ["*.example.com"]`,
		``,
		`[proxies.plugin]`,
		`type = "http2http"`,
		`localAddr = "traefik.traefik.svc:80"`,
	}, "\n")

	if !strings.Contains(configuration, expected) {
		t.Fatalf("expected configuration to contain:\n%s\ngot:\n%s", expected, configuration)
	}
}

func TestCreateConfigurationRendersHTTPProxyWithServiceRef(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "http-tunnel"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			HTTP: &frpv1.HTTP{
				CustomDomains: []string{"app.example.com", "www.example.com"},
				ServiceRef:    &frpv1.ServiceRef{Name: "web"},
				LocalPort:     ptr(8080),
			},
			Transport: &frpv1.Transport{UseCompression: true},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`[[proxies]]`,
		`name = "http-tunnel"`,
		`type = "http"`,
		`customDomains = ["app.example.com", "www.example.com"]`,
		`localIP = "web"`,
		`localPort = 8080`,
		`transport.useCompression = true`,
	}, "\n")

	if !strings.Contains(configuration, expected) {
		t.Fatalf("expected configuration to contain:\n%s\ngot:\n%s", expected, configuration)
	}
}

func TestCreateConfigurationEscapesTokenAndHost(t *testing.T) {
	server := exitServer()
	server.Spec.Host = `frps."evil".org`

	configuration, err := CreateConfiguration(server, `to"ken`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(configuration, `serverAddr = "frps.\"evil\".org"`) {
		t.Fatalf("expected escaped host, got:\n%s", configuration)
	}
	if !strings.Contains(configuration, `auth.token = "to\"ken"`) {
		t.Fatalf("expected escaped token, got:\n%s", configuration)
	}
}

func TestCreateConfigurationRendersHTTPRoutingOptions(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "http-routing"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			HTTP: &frpv1.HTTP{
				Subdomain:         ptr("portal"),
				Locations:         []string{"/", "/api"},
				HTTPUser:          ptr("alice"),
				HTTPPassword:      ptr(`p"ass`),
				HostHeaderRewrite: ptr("backend.example.com"),
				RouteByHTTPUser:   ptr("alice"),
				RequestHeaders: map[string]string{
					"x-from-where": "frp",
					"x-trace":      `a"b`,
				},
				ResponseHeaders: map[string]string{"x-proxy": "frp"},
				ServiceRef:      &frpv1.ServiceRef{Name: "web", Namespace: ptr("apps")},
				LocalPort:       ptr(8080),
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedLines := []string{
		`subdomain = "portal"`,
		`locations = ["/", "/api"]`,
		`httpUser = "alice"`,
		`httpPassword = "p\"ass"`,
		`hostHeaderRewrite = "backend.example.com"`,
		`routeByHTTPUser = "alice"`,
		`requestHeaders.set."x-from-where" = "frp"`,
		`requestHeaders.set."x-trace" = "a\"b"`,
		`responseHeaders.set."x-proxy" = "frp"`,
		`localIP = "web.apps.svc"`,
		`localPort = 8080`,
	}
	for _, expected := range expectedLines {
		if !strings.Contains(configuration, expected) {
			t.Errorf("expected configuration to contain %q, got:\n%s", expected, configuration)
		}
	}
	if strings.Contains(configuration, "customDomains =") {
		t.Errorf("did not expect customDomains for a subdomain-only proxy, got:\n%s", configuration)
	}
}

func TestCreateConfigurationRendersPluginOptions(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			HTTP: &frpv1.HTTP{
				CustomDomains: []string{"plugin.example.com"},
				Plugin: &frpv1.Plugin{
					Type:              "http2https",
					ServiceRef:        frpv1.ServiceRef{Name: "secure-web"},
					LocalPort:         8443,
					HostHeaderRewrite: ptr("origin.example.com"),
					RequestHeaders:    map[string]string{"x-forwarded-by": "frp"},
				},
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`[proxies.plugin]`,
		`type = "http2https"`,
		`localAddr = "secure-web:8443"`,
		`hostHeaderRewrite = "origin.example.com"`,
		`requestHeaders.set."x-forwarded-by" = "frp"`,
	}, "\n")
	if !strings.Contains(configuration, expected) {
		t.Fatalf("expected configuration to contain:\n%s\ngot:\n%s", expected, configuration)
	}
	if strings.Contains(configuration, "localIP =") || strings.Contains(configuration, "localPort =") {
		t.Fatalf("plugin proxy must not render plain backend fields, got:\n%s", configuration)
	}
}

func TestCreateConfigurationRendersAllTransportOptions(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "transport"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			TCP: &frpv1.TCP{
				ServiceRef: frpv1.ServiceRef{Name: "service"},
				LocalPort:  8080,
				RemotePort: 18080,
			},
			Transport: &frpv1.Transport{
				UseEncryption:      true,
				UseCompression:     true,
				ProxyProtocol:      ptr("v2"),
				BandwidthLimit:     ptr("10MB"),
				BandwidthLimitMode: ptr("client"),
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, expected := range []string{
		`transport.useEncryption = true`,
		`transport.useCompression = true`,
		`transport.proxyProtocolVersion = "v2"`,
		`transport.bandwidthLimit = "10MB"`,
		`transport.bandwidthLimitMode = "client"`,
	} {
		if !strings.Contains(configuration, expected) {
			t.Errorf("expected configuration to contain %q, got:\n%s", expected, configuration)
		}
	}
}

func TestCreateConfigurationSortsProxiesByName(t *testing.T) {
	newTunnel := func(name string) frpv1.Tunnel {
		return frpv1.Tunnel{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: frpv1.TunnelSpec{
				ExitServer: "guest-frps",
				TCP:        &frpv1.TCP{ServiceRef: frpv1.ServiceRef{Name: "service"}, LocalPort: 80, RemotePort: 80},
			},
		}
	}

	configuration, err := CreateConfiguration(exitServer(), "token", []frpv1.Tunnel{newTunnel("z-last"), newTunnel("a-first")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	first := strings.Index(configuration, `name = "a-first"`)
	last := strings.Index(configuration, `name = "z-last"`)
	if first < 0 || last < 0 || first >= last {
		t.Fatalf("expected proxies sorted by name, got:\n%s", configuration)
	}
}
