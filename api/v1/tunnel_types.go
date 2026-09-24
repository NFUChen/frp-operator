/*
Copyright 2026 Aureum Cloud, N-Bit, Niek Berenschot.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TunnelSpec defines the desired state of Tunnel
// +kubebuilder:validation:XValidation:rule="has(self.tcp) != has(self.http)",message="exactly one of tcp or http must be set"
type TunnelSpec struct {
	ExitServer string `json:"exitServer"`

	// +optional
	TCP *TCP `json:"tcp,omitempty"`

	// +optional
	HTTP *HTTP `json:"http,omitempty"`

	// +optional
	Transport *Transport `json:"transport,omitempty"`
}

type TCP struct {
	ServiceRef ServiceRef `json:"serviceRef"`
	LocalPort  int        `json:"localPort"`
	RemotePort int        `json:"remotePort"`
}

// HTTP exposes the tunnel as a vhost based HTTP proxy on the exit server.
// +kubebuilder:validation:XValidation:rule="has(self.plugin) != (has(self.serviceRef) && has(self.localPort))",message="either plugin or both serviceRef and localPort must be set"
// +kubebuilder:validation:XValidation:rule="size(self.customDomains) > 0 || has(self.subdomain)",message="at least one custom domain or a subdomain must be set"
type HTTP struct {
	// +optional
	CustomDomains []string `json:"customDomains,omitempty"`

	// +optional
	Subdomain *string `json:"subdomain,omitempty"`

	// +optional
	Locations []string `json:"locations,omitempty"`

	// +optional
	HTTPUser *string `json:"httpUser,omitempty"`

	// +optional
	HTTPPassword *string `json:"httpPassword,omitempty"`

	// +optional
	HostHeaderRewrite *string `json:"hostHeaderRewrite,omitempty"`

	// +optional
	RequestHeaders map[string]string `json:"requestHeaders,omitempty"`

	// +optional
	ResponseHeaders map[string]string `json:"responseHeaders,omitempty"`

	// +optional
	RouteByHTTPUser *string `json:"routeByHTTPUser,omitempty"`

	// +optional
	ServiceRef *ServiceRef `json:"serviceRef,omitempty"`

	// +optional
	LocalPort *int `json:"localPort,omitempty"`

	// +optional
	Plugin *Plugin `json:"plugin,omitempty"`
}

// Plugin configures an frpc client plugin instead of a plain local address.
type Plugin struct {
	// +kubebuilder:validation:Enum=http2http;http2https
	Type string `json:"type"`

	ServiceRef ServiceRef `json:"serviceRef"`
	LocalPort  int        `json:"localPort"`

	// +optional
	HostHeaderRewrite *string `json:"hostHeaderRewrite,omitempty"`

	// +optional
	RequestHeaders map[string]string `json:"requestHeaders,omitempty"`
}

type ServiceRef struct {
	Name      string  `json:"name"`
	Namespace *string `json:"namespace,omitempty"`
}

type Transport struct {
	// +optional
	UseEncryption bool `json:"useEncryption,omitempty"`

	// +optional
	UseCompression bool `json:"useCompression,omitempty"`

	// +kubebuilder:validation:Enum=v1;v2
	// +optional
	ProxyProtocol *string `json:"proxyProtocol,omitempty"`

	// +kubebuilder:validation:Pattern=^\d+(KB|MB)$
	// +optional
	BandwidthLimit *string `json:"bandwidthLimit,omitempty"`

	// +kubebuilder:validation:Enum=client;server
	// +optional
	BandwidthLimitMode *string `json:"bandwidthLimitMode,omitempty"`
}

// TunnelStatus defines the observed state of Tunnel
type TunnelStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Tunnel is the Schema for the tunnels API
type Tunnel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TunnelSpec   `json:"spec,omitempty"`
	Status TunnelStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TunnelList contains a list of Tunnel
type TunnelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tunnel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tunnel{}, &TunnelList{})
}
