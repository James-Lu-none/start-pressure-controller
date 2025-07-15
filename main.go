package main

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	threshold   = 5
	taintKey    = "start-pressure"
	taintEffect = v1.TaintEffectNoSchedule
)

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	for {
		nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), meta.ListOptions{})
		if err != nil {
			fmt.Println("Error listing nodes:", err)
			continue
		}

		for _, node := range nodes.Items {
			pods, err := clientset.CoreV1().Pods("").List(context.TODO(), meta.ListOptions{
				FieldSelector: fmt.Sprintf("spec.nodeName=%s", node.Name),
			})
			if err != nil {
				fmt.Printf("Error listing pods on node %s: %v\n", node.Name, err)
				continue
			}

			startingCount := 0
			for _, pod := range pods.Items {
				if pod.Status.Phase == v1.PodPending {
					startingCount++
					continue
				}

				if pod.Status.Phase == v1.PodRunning {
					ready := false
					for _, cond := range pod.Status.Conditions {
						if cond.Type == v1.PodReady && cond.Status == v1.ConditionTrue {
							ready = true
						}
					}
					if !ready {
						startingCount++
					}
				}
			}

			taintPresent := false
			for _, t := range node.Spec.Taints {
				if t.Key == taintKey && t.Effect == taintEffect {
					taintPresent = true
					break
				}
			}

			if startingCount > threshold && !taintPresent {
				fmt.Printf("Tainting node %s (starting pods: %d)\n", node.Name, startingCount)
				node.Spec.Taints = append(node.Spec.Taints, v1.Taint{
					Key:    taintKey,
					Value:  "high",
					Effect: taintEffect,
				})
				_, err := clientset.CoreV1().Nodes().Update(context.TODO(), &node, meta.UpdateOptions{})
				if err != nil {
					fmt.Printf("Failed to taint node %s: %v\n", node.Name, err)
				}
			} else if startingCount <= threshold && taintPresent {
				newTaints := []v1.Taint{}
				for _, t := range node.Spec.Taints {
					if t.Key != taintKey {
						newTaints = append(newTaints, t)
					}
				}
				fmt.Printf("Removing taint from node %s (starting pods: %d)\n", node.Name, startingCount)
				node.Spec.Taints = newTaints
				_, err := clientset.CoreV1().Nodes().Update(context.TODO(), &node, meta.UpdateOptions{})
				if err != nil {
					fmt.Printf("Failed to remove taint from node %s: %v\n", node.Name, err)
				}
			}
		}

		time.Sleep(30 * time.Second)
	}
}
