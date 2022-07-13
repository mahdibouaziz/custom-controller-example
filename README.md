# custom-controller-example

## Requirements

You must read the Docs that I've made of the client-go

## Note:

we use tools like `kubebuilder` or `operator sdk` which **generates a controller scaffold for us**, but in this demo we will write everything from scratch


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

we just add this code
```go
	// creating a shared informer factory
	informers := informers.NewSharedInformerFactory(clientset, 30*time.Second)
```

## Creating Controller type


create a controller.go file

`touch controller.go`

now we created a basic skeleton of the controller

```go
package main

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type controller struct {
	// a clientset to interact with K8s cluster
	clientset kubernetes.Interface

	// our lister
	depLister appslisters.DeploymentLister

	// check if the cache is synced or not
	depCacheSyncd cache.InformerSynced

	// our queue
	queue workqueue.RateLimitingInterface
}

func newController(clientset kubernetes.Interface, depInformer appsinformers.DeploymentInformer) *controller {
	c := &controller{
		clientset:     clientset,
		depLister:     depInformer.Lister(),
		depCacheSyncd: depInformer.Informer().HasSynced,
		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ekpose"),
	}

	// register the handlers
	depInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    handleAdd,
		DeleteFunc: handleDel,
	})

	return c
}

func (c *controller) run(ch <-chan struct{}) {
	fmt.Println("Starting controller")
	// we need to wait the informer local caches
	if !cache.WaitForCacheSync(ch, c.depCacheSyncd) {
		fmt.Printf("Waiting for cache to be synced\n")
	}

	// run the funciton worker every period of time until the channel ch is closed
	go wait.Until(c.worker, 1*time.Second, ch)

	<-ch
}

func (c *controller) worker() {
	//
}

func handleAdd(obj interface{}) {
	fmt.Println("add was called")
}

func handleDel(obj interface{}) {
	fmt.Println("delete was called")
}
```

and in the `main.go` file we add:

```go
    ch := make(chan struct{})
	c := newController(clientset, informers.Apps().V1().Deployments())
	informers.Start(ch)
	c.run(ch)
```

## Addeding the objects in the Queue

in the handle functions we just add `c.queue.Add(obj)`

```go
func (c *controller) handleAdd(obj interface{}) {
	fmt.Println("add was called")
	c.queue.Add(obj)
}

func (c *controller) handleDel(obj interface{}) {
	fmt.Println("delete was called")
	c.queue.Add(obj)
}
```

go read the code:
vide reference [https://www.youtube.com/watch?v=lzoWSfvE2yA&list=PLh4KH3LtJvRQ43JAwwjvTnsVOMp0WKnJO&ab_channel=VivekSingh]

and to test it just run the ekpose and create an nginx deployment !! and then a service will be created automatically

you can see it if you do port forwarding `k port-forward -n ekposetest svc/nginx 8080:80`