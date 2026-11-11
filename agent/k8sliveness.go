package main

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// runK8sLiveness watches pods scheduled on this node and reports each
// workload's liveness under the same service name runKubernetes uses for its
// logs, via serviceFromPod. It needs RBAC to get/list/watch pods (see
// agent/deploy/daemonset.yaml); if that is missing it logs and backs off
// rather than crashing, so a cluster with no RBAC still ships logs fine, just
// without liveness.
func runK8sLiveness(ctx context.Context, report *LivenessReporter) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("argus-agent: k8s liveness: not running in a cluster, skipping: %v", err)
		return
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Printf("argus-agent: k8s liveness: build client: %v", err)
		return
	}
	node := os.Getenv("ARGUS_NODE_NAME")
	if node == "" {
		log.Print("argus-agent: k8s liveness: ARGUS_NODE_NAME is not set (see daemonset.yaml), skipping")
		return
	}

	roll := &podRollup{ready: map[string]bool{}}
	backoff := time.Second
	for ctx.Err() == nil {
		started := time.Now()
		if err := watchNodePods(ctx, clientset, node, roll, report); err != nil && ctx.Err() == nil {
			log.Printf("argus-agent: k8s liveness: %v", err)
		}
		if time.Since(started) > time.Minute {
			backoff = time.Second
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

// watchNodePods lists then watches this node's pods until the watch ends or
// ctx is cancelled, reporting liveness on every add/modify/delete.
func watchNodePods(ctx context.Context, cs kubernetes.Interface, node string, roll *podRollup, report *LivenessReporter) error {
	sel := "spec.nodeName=" + node
	list, err := cs.CoreV1().Pods("").List(ctx, metav1.ListOptions{FieldSelector: sel})
	if err != nil {
		return err
	}
	for i := range list.Items {
		roll.update(&list.Items[i], report)
	}
	w, err := cs.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{FieldSelector: sel, ResourceVersion: list.ResourceVersion})
	if err != nil {
		return err
	}
	defer w.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-w.ResultChan():
			if !ok {
				return errors.New("watch channel closed")
			}
			pod, ok := ev.Object.(*corev1.Pod)
			if !ok {
				continue
			}
			if ev.Type == watch.Deleted {
				roll.remove(pod.Name, report)
				continue
			}
			roll.update(pod, report)
		}
	}
}

// podRollup aggregates per-pod readiness to per-service liveness: a service
// (workload) counts as alive if any of its replicas on this node is ready,
// so one crash-looping replica does not flip a healthy service to "down".
type podRollup struct {
	mu    sync.Mutex
	ready map[string]bool // pod name -> ready
}

func (r *podRollup) update(p *corev1.Pod, report *LivenessReporter) {
	r.mu.Lock()
	r.ready[p.Name] = p.Status.Phase == corev1.PodRunning && podReady(p)
	report.Report(serviceFromPod(p.Name), "kubernetes", r.anyReady(serviceFromPod(p.Name)), string(p.Status.Phase))
	r.mu.Unlock()
}

func (r *podRollup) remove(podName string, report *LivenessReporter) {
	r.mu.Lock()
	delete(r.ready, podName)
	svc := serviceFromPod(podName)
	report.Report(svc, "kubernetes", r.anyReady(svc), "deleted")
	r.mu.Unlock()
}

// anyReady reports whether any tracked pod of svc is ready. Callers hold r.mu.
func (r *podRollup) anyReady(svc string) bool {
	for name, ok := range r.ready {
		if ok && serviceFromPod(name) == svc {
			return true
		}
	}
	return false
}

func podReady(p *corev1.Pod) bool {
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}
