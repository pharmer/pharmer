package cloud

import (
	"fmt"
	mrnd "math/rand"
	"time"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/api"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/fields"
)

func GetKubeadmToken() string {
	return fmt.Sprintf("%s.%s", RandStringRunes(6), RandStringRunes(16))
}

func init() {
	mrnd.Seed(time.Now().UnixNano())
}

// Hexidecimal
var letterRunes = []rune("0123456789abcdef")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[mrnd.Intn(len(letterRunes))]
	}
	return string(b)
}

func CheckValidToken(kc kubernetes.Interface)  {
	secrets, err := kc.CoreV1().Secrets(metav1.NamespaceSystem).List(metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(map[string]string{
			api.SecretTypeField: "bootstrap.kubernetes.io/token",
		}).String(),
	})
	fmt.Println(secrets, err)
}