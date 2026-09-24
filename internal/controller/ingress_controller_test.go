package controller

import (
	"context"
	"strings"
	"testing"

	frpv1 "frp-operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestIngressReconcilerCreatesTunnel(t *testing.T) {
	scheme := testScheme(t)
	className := "frp"
	pathType := networkingv1.PathTypePrefix
	objects := []runtime.Object{
		&networkingv1.IngressClass{ObjectMeta: metav1.ObjectMeta{Name: className}, Spec: networkingv1.IngressClassSpec{Controller: IngressControllerName}},
		&frpv1.ExitServer{ObjectMeta: metav1.ObjectMeta{Name: "public", Namespace: "apps"}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "apps"}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Name: "http", Port: 8080}}}},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "apps", Annotations: map[string]string{ExitServerAnnotation: "public"}},
			Spec:       networkingv1.IngressSpec{IngressClassName: &className, Rules: []networkingv1.IngressRule{{Host: "web.example.com", IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{Path: "/api", PathType: &pathType, Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "web", Port: networkingv1.ServiceBackendPort{Name: "http"}}}}}}}}}},
		},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()
	r := &IngressReconciler{Client: client, Scheme: scheme}

	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "web", Namespace: "apps"}}); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	list := &frpv1.TunnelList{}
	if err := client.List(context.Background(), list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("got %d tunnels, want 1", len(list.Items))
	}
	tunnel := list.Items[0]
	if tunnel.Spec.ExitServer != "public" || tunnel.Spec.HTTP == nil {
		t.Fatalf("unexpected tunnel spec: %#v", tunnel.Spec)
	}
	if got := *tunnel.Spec.HTTP.LocalPort; got != 8080 {
		t.Errorf("local port = %d, want 8080", got)
	}
	if got := tunnel.Spec.HTTP.Locations[0]; got != "/api" {
		t.Errorf("location = %q, want /api", got)
	}
	if !metav1.IsControlledBy(&tunnel, objects[3].(*networkingv1.Ingress)) {
		t.Error("tunnel is not controlled by ingress")
	}
}

func TestIngressReconcilerIgnoresOtherClass(t *testing.T) {
	scheme := testScheme(t)
	className := "other"
	ingress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "apps"}, Spec: networkingv1.IngressSpec{IngressClassName: &className}}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		ingress,
		&networkingv1.IngressClass{ObjectMeta: metav1.ObjectMeta{Name: className}, Spec: networkingv1.IngressClassSpec{Controller: "example.com/other"}},
	).Build()
	r := &IngressReconciler{Client: client, Scheme: scheme}

	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "web", Namespace: "apps"}}); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	list := &frpv1.TunnelList{}
	_ = client.List(context.Background(), list)
	if len(list.Items) != 0 {
		t.Fatalf("got %d tunnels, want 0", len(list.Items))
	}
}

func TestIngressReconcilerRejectsTLS(t *testing.T) {
	scheme := testScheme(t)
	className := "frp"
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "apps", Annotations: map[string]string{ExitServerAnnotation: "public"}},
		Spec:       networkingv1.IngressSpec{IngressClassName: &className, TLS: []networkingv1.IngressTLS{{Hosts: []string{"web.example.com"}}}},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		ingress,
		&networkingv1.IngressClass{ObjectMeta: metav1.ObjectMeta{Name: className}, Spec: networkingv1.IngressClassSpec{Controller: IngressControllerName}},
	).Build()
	r := &IngressReconciler{Client: client, Scheme: scheme}

	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "web", Namespace: "apps"}}); err == nil {
		t.Fatal("Reconcile() error = nil, want TLS unsupported error")
	}
}

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := networkingv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := frpv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return scheme
}
