package main

import (
	"fmt"
	"os"
	"os/user"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"encoding/json"
	"log"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func watchForServices(timeout int64, nodeAddress string, clientset *kubernetes.Clientset) error {
	watchServicesInterface, err := clientset.Core().Services("").Watch(metav1.ListOptions{
		Watch:          true,
		TimeoutSeconds: &timeout,
	})
	if err != nil {
		log.Printf("Error retrieving watch interface for services: %+v", err)
		panic(err.Error())
	}

	events := watchServicesInterface.ResultChan()
	for {
		event, ok := <-events
		if event.Type == "ADDED" {
			var svc = v1.Service{}
			b, err := json.Marshal(event.Object)
			if err = json.Unmarshal(b, &svc); err == nil {
				if svc.Spec.Type == "LoadBalancer" && len(svc.Status.LoadBalancer.Ingress) == 0 {
					fmt.Printf("Service %q has no ingress for its loadbalancer, updating to %s\n", svc.Name, nodeAddress)
					patch := []byte(fmt.Sprintf(`[{"op": "add", "path": "/status/loadBalancer/ingress", "value":  [ { "ip": "%s" } ] }]`, nodeAddress))
					err := clientset.CoreV1().RESTClient().Patch(types.JSONPatchType).Resource("services").Namespace(svc.Namespace).Name(svc.Name).SubResource("status").Body(patch).Do().Error()
					if err != nil {
						log.Fatalf("Error patching service %s: %s", svc.Name, err)
					}
				}
			}
		}

		if !ok {
			break
		}
	}
	return err
}

func main() {
	kubernetes_host := os.Getenv("KUBERNETES_SERVICE_HOST")
	kubeconfig := os.Getenv("KUBECONFIG")

	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Error getting current user: %s", err)
	}

	if kubeconfig == "" {
		kubeconfig = usr.HomeDir + "/.kube/config"
	}

	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		fmt.Printf("kubeconfig not found at %v\n", kubeconfig)
		kubeconfig = ""
	}

	var config *rest.Config

	if kubernetes_host != "" {
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalf("Error configuring in-cluster client: %s", err)
		}
	} else if kubeconfig != "" {
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.Fatalf("Error configuring kubeconfig client: %s", err)
		}
	} else {
		config = &rest.Config{
			Host: "http://127.0.0.1:8001",
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error preparing connection to kubernetes cluster: %s", err)
	}

	node, err := clientset.CoreV1().Nodes().Get("minikube", metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Error getting minikube worker node: %s", err)
	}

	watchTimeout := int64(1800)

	log.Printf("%s", node.Status.Addresses[0].Address)

	for {
		go watchForServices(watchTimeout, node.Status.Addresses[0].Address, clientset)
		time.Sleep(time.Second*time.Duration(watchTimeout) + time.Second*2)
		log.Printf("Restarting watchers from %v second timeout.", watchTimeout+2)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
