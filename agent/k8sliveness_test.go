package main

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func pod(name string, phase corev1.PodPhase, ready bool) *corev1.Pod {
	status := corev1.ConditionFalse
	if ready {
		status = corev1.ConditionTrue
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.PodStatus{
			Phase:      phase,
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: status}},
		},
	}
}

func TestPodRollupServiceNaming(t *testing.T) {
	roll := &podRollup{ready: map[string]bool{}}
	r := NewLivenessReporter("http://x", "k")

	// api-7d9f8b6c5d-x2k4p and api-7d9f8b6c5d-y3n5q both roll up to service "api".
	roll.update(pod("api-7d9f8b6c5d-x2k4p", corev1.PodRunning, true), r)
	if p := r.pending["api"]; !p.Alive {
		t.Fatalf("service api: want alive, got %+v", p)
	}
}

func TestPodRollupAnyReplicaReadyKeepsServiceAlive(t *testing.T) {
	roll := &podRollup{ready: map[string]bool{}}
	r := NewLivenessReporter("http://x", "k")

	roll.update(pod("api-7d9f8b6c5d-x2k4p", corev1.PodRunning, true), r)
	roll.update(pod("api-7d9f8b6c5d-y3n5q", corev1.PodRunning, false), r) // crash-looping replica

	if p := r.pending["api"]; !p.Alive {
		t.Fatalf("one ready replica should keep the service alive, got %+v", p)
	}
}

func TestPodRollupDownWhenAllReplicasUnready(t *testing.T) {
	roll := &podRollup{ready: map[string]bool{}}
	r := NewLivenessReporter("http://x", "k")

	roll.update(pod("api-7d9f8b6c5d-x2k4p", corev1.PodRunning, false), r)
	if p := r.pending["api"]; p.Alive {
		t.Fatalf("no ready replicas: want down, got %+v", p)
	}
}

func TestPodRollupRemoveDropsPodFromAggregate(t *testing.T) {
	roll := &podRollup{ready: map[string]bool{}}
	r := NewLivenessReporter("http://x", "k")

	roll.update(pod("api-7d9f8b6c5d-x2k4p", corev1.PodRunning, true), r)
	roll.remove("api-7d9f8b6c5d-x2k4p", r)

	if p := r.pending["api"]; p.Alive {
		t.Fatalf("after removing the only ready replica: want down, got %+v", p)
	}
}
