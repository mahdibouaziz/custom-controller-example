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
