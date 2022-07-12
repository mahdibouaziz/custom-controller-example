# custom-controller-example

## Requirements

You must read the Docs that I've made of the client-go

## Use case of our custom controller

We are going to create a custom controller called **ekpose** that will create a `service` and an `ingress` resource as soon as a `deployment` is created.

## Basic building blocks of this controller

1. The controller should get to know when a particular deployment resource is created and only after that it should expose that deployment resource ==> we need to `watch` our cluster. But in prod we should not use `watch`, instead we should use `informers`.
2. We need to **register** some functions to the `informer` so that as soon as a deployment is **added**, **deleted** or **updated** on the cluster, the **appropriate function should be called**.
3. We must `enqueue` the function call to the `queue` with the right parameters.
4. Another `go routine` should start to run all the bussiness logic of the function call (it must create service and ingress to the deployment in our case)

# Start the demo:

## Setup the basic go project (create a clientset)

create a go module named ekpose

`go mod init ekpose`

create a main.go file

`touch main.go`

And paste this code (just creating a basic clientset)

```go
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

```

## Creating a SharedInformerFactory


