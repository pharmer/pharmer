package inspector

import (
	"fmt"

	"github.com/appscode/go/errors"
	term "github.com/appscode/go-term"
	"github.com/cenkalti/backoff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	apiv1 "k8s.io/api/core/v1"
)

func (i *Inspector) CheckHelthStatus() error {
	term.Println("Checking for component status...")
	backoff.Retry(func() error {
		resp, err := i.client.CoreV1().ComponentStatuses().List(metav1.ListOptions{
			LabelSelector: labels.Everything().String(),
		})
		if err != nil {
			return err
		}
		for _, status := range resp.Items {
			for _, cond := range status.Conditions {
				if cond.Type == apiv1.ComponentHealthy && cond.Status != apiv1.ConditionTrue {
					return errors.New().WithMessagef("Component %v is in condition %v with status %v", status.Name, cond.Type, cond.Status).Err()
				} else {
					term.Infoln(fmt.Sprintf("Component %v is in condition %v with status %v", status.Name, cond.Type, cond.Status))
				}
			}
		}
		return nil
	}, backoff.NewExponentialBackOff())

	term.Successln("Component status are ok")
	return nil
}

func (i *Inspector) checkRBAC() error {
	if _, err := i.client.Discovery().ServerResourcesForGroupVersion("authentication.k8s.io/v1beta1"); err != nil {
		term.Errorln("RBAC authentication is not enabled")
		return errors.FromErr(err).Err()
	} else {
		term.Successln("RBAC authentication is enabled")
	}
	if _, err := i.client.Discovery().ServerResourcesForGroupVersion("authorization.k8s.io/v1beta1"); err != nil {
		term.Errorln("RBAC authorization is not enabled")
		return errors.FromErr(err).Err()
	} else {
		term.Successln("RBAC authorization is enabled")
	}
	return nil
}

/*func (i *Inspector) checkLoadBalancer() error {
	retry := 5
	for retry > 0 {
		if _, err := c.Kube.VoyagerClient.Ingresses("appscode").Get("default-lb", metav1.GetOptions{}); err != nil {
			term.Errorln("Default load balancer is not set up")
		} else {
			term.Successln("Default load balancer is set up")
			break
		}
		time.Sleep(15 * time.Second)
		fmt.Println("Retrying...")
		retry--
	}
	retry = 5
	flag := false
	for retry > 0 {
		pods, err := c.Kube.Client.CoreV1().Pods("appscode").List(metav1.ListOptions{})
		if err != nil {
			term.Errorln("Default load balancer pod is not set up")
		}
		for _, p := range pods.Items {
			if strings.HasPrefix(p.Name, "voyager-default-lb") {
				flag = true
				break
			}
		}
		if flag {
			break
		}
		time.Sleep(15 * time.Second)
		fmt.Println("Retrying...")
		retry--
	}
	if !flag {
		term.Errorln("Default load balancer pod is not found")
	} else {
		term.Successln("Default load balancer pod is found")
	}

	return nil
}*/
