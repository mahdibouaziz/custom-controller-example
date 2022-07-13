package main

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		AddFunc:    c.handleAdd,
		DeleteFunc: c.handleDel,
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
	for c.processItem() {

	}
}

// getting the object from the queue and do the work
func (c *controller) processItem() bool {
	// get the item from the queue if it exists
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}

	defer c.queue.Forget(item)
	// Get the key of the item (it contains the namespance and the name)
	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		fmt.Printf("getting key from chache %s \n", err.Error())
		return false
	}

	// get the namespace and the name of the object
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		fmt.Printf("splitting key into ns and name %s \n", err.Error())
		return false
	}

	// check if the object has been deleted from  K8s cluster (query the API-Server)
	ctx := context.TODO()
	_, err = c.clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		fmt.Printf("handle delete event for deployment %s\n", name)
		// delete the service
		// TODO: in a prod level you must have a real delete logic (by annotaion or by owner)
		err = c.clientset.CoreV1().Services(ns).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			fmt.Printf("deleting service %s ,error %s\n", name, err.Error())
			return false
		}

		// delete ingress
		// TODO: in a prod level you must have a real delete logic (by annotaion or by owner)
		err = c.clientset.NetworkingV1().Ingresses(ns).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			fmt.Printf("deleting ingress %s ,error %s\n", name, err.Error())
			return false
		}

		return true
	}

	// create a service and ingress for this deployment and check for failure cases
	err = c.syncDeployment(ns, name)
	if err != nil {
		// retry if it fails
		fmt.Printf("syncing deployment %s\n", err.Error())
		return false
	}

	return true
}

func (c *controller) syncDeployment(ns string, name string) error {
	ctx := context.TODO()

	// we used it in the name of the service
	dep, err := c.depLister.Deployments(ns).Get(name)
	if err != nil {
		fmt.Printf("Getting deployment from lister %s\n", err.Error())
	}

	// we used them in the selector of the service
	labels := deplLabels(dep)
	// port := deplContainerPod()

	// create a service with the configuration needed
	// TODO: we have to modify this to figure out the port of our deployment container is listening
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dep.Name,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}
	s, err := c.clientset.CoreV1().Services(ns).Create(ctx, &svc, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("creating service %s\n", err.Error())
	}

	// create ingress

	return createIngress(ctx, c.clientset, s)
}

func createIngress(ctx context.Context, client kubernetes.Interface, svc *corev1.Service) error {
	pathType := "Prefix"
	// TODO: we have to modify this to figure out the port of our service
	ingress := netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/",
			},
		},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{
				{
					IngressRuleValue: netv1.IngressRuleValue{
						HTTP: &netv1.HTTPIngressRuleValue{
							Paths: []netv1.HTTPIngressPath{
								{
									Path:     fmt.Sprintf("/%s", svc.Name),
									PathType: (*netv1.PathType)(&pathType),
									Backend: netv1.IngressBackend{
										Service: &netv1.IngressServiceBackend{
											Name: svc.Name,
											Port: netv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	// create the ingress resource
	_, err := client.NetworkingV1().Ingresses(svc.Namespace).Create(ctx, &ingress, metav1.CreateOptions{})
	return err
}

func deplLabels(dep *appsv1.Deployment) map[string]string {
	return dep.Spec.Template.Labels
}

func (c *controller) handleAdd(obj interface{}) {
	fmt.Println("add was called")
	c.queue.Add(obj)
}

func (c *controller) handleDel(obj interface{}) {
	fmt.Println("delete was called")
	c.queue.Add(obj)
}
