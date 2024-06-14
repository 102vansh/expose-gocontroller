package main

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/wait"
	appinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	applisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type controller struct {
	clientset    kubernetes.Interface
	deplister    applisters.DeploymentLister
	depcachesync cache.InformerSynced
	queue        workqueue.RateLimitingInterface
}

func newController(clientset kubernetes.Interface, depInformer appinformers.DeploymentInformer) *controller {
	c := &controller{
		clientset:    clientset,
		deplister:    depInformer.Lister(),
		depcachesync: depInformer.Informer().HasSynced,
		queue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
	depInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleAdd,
			DeleteFunc: c.handleDel,
		},
	)

	return c
}
func (c *controller) run(ch <-chan struct{}) {
	fmt.Println("controller is starting")
	if !cache.WaitForCacheSync(ch, c.depcachesync) {
		fmt.Println("waiting for cache sync")
	}
	go wait.Until(c.worker, 1*time.Second, ch)
	<-ch
}
func (c *controller) worker() {
	for c.processItem() {

	}
}
func (c *controller) processItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Forget(item)
	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		fmt.Printf("getting key%s\n", err.Error())
	}
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		fmt.Println("splitting key into namespace and name", err.Error())
	}
	err = c.syncDeployment(ns, name)
	if err != nil {
		//retry

		fmt.Printf("sync deploy%s\n", err.Error())
		return false
	}
	return true
}
func (c *controller) syncDeployment(ns, name string) error {

	dep, err := c.deplister.Deployments(ns).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Printf("Deployment %s in work queue no longer exists\n")
			return nil
		}
		return err
	}
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dep.Name,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Selector: depLabels(*dep),
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Name: "http",
					Port: 80,
				},
			},
		},
	}
	_, err = c.clientset.CoreV1().Services(ns).Create(context.Background(), &svc, metav1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}
	return nil
}
func depLabels(dep appsv1.Deployment) map[string]string {
	return dep.Spec.Template.Labels
}
func (c *controller) handleAdd(obj interface{}) {
	fmt.Println("add is called")
	c.queue.Add(obj)
}
func (c *controller) handleDel(obj interface{}) {
	fmt.Println("delete is called")
	c.queue.Add(obj)
}
