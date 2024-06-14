package main

import (
	"fmt"
	"os"
	"time"

	"path/filepath"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err.Error())
	}

	kubeconfigpath := filepath.Join(home, ".kube/config")
	fmt.Println(kubeconfigpath)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigpath)
	if err != nil {
		panic(err.Error())
	}
	client, err := kubernetes.NewForConfig(config)

	if err != nil {
		panic(err.Error())
	}
	ch := make(chan struct{})
	informer := informers.NewSharedInformerFactory(client, 10*time.Minute)
	c := newController(client, informer.Apps().V1().Deployments())
	informer.Start(ch)
	c.run(ch)

	fmt.Println(informer)
}
