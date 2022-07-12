package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeconfig := flag.String("kubeconfig", filepath.Join(os.Getenv("HOME"), ".kube", "config"), "Location to your kubeconfig file")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		// handle error
		fmt.Printf("error %s building config from flags", err.Error())
		config, err = rest.InClusterConfig()
		if err != nil {
			fmt.Printf("error %s getting inclusterconfig", err.Error())
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("error %s creating a clientset", err.Error())
	}

	fmt.Println(clientset)

}
