package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/peterh/liner"

	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	line := liner.NewLiner()
	defer line.Close()
	line.SetCtrlCAborts(true)

	// 获取配置文件路径
	kubeconfig := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	namespace := flag.String("namespace", "default", "namespace")
	flag.Parse()
	// 配置文件不能为空
	if *kubeconfig == "" {
		fmt.Println("Kubeconfig file is required")
		return
	}

	// 使用配置文件创建k8s客户端
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Printf("Error building kubeconfig: %v\n", err)
		return
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Error creating Kubernetes client: %v\n", err)
		return
	}

	// 获取Pod列表
	pods, err := clientset.CoreV1().Pods(*namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error listing pods: %v\n", err)
		return
	}

	// 显示Pod列表并标号
	fmt.Println("Pods in namespace", *namespace)
	for i, pod := range pods.Items {
		fmt.Printf("[\u001B[1;31m %d \u001B[0m] %s \u001B[0;32m%s\u001B[0m \n", i, pod.Name, pod.Status.Phase)
	}

	for {

		fmt.Println("Enter exit to quit")
		// 用户输入编号 或者 继续搜索
		//reader := bufio.NewReader(os.Stdin)
		//fmt.Print("Enter pod number or search: ")

		input, err := line.Prompt("Enter pod number or search: ")
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			return
		}
		//input, _ = reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			line.AppendHistory(input)
		}

		// 检查输入是否为数字
		podNumber, err := strconv.Atoi(input)
		if err == nil && podNumber >= 0 && podNumber < len(pods.Items) {
			selectedPod := pods.Items[podNumber]
			handlePodAction(line, selectedPod, kubeconfig, namespace)
		} else {
			//如果== exit 退出
			if input == "exit" {
				fmt.Println("bye bye")
				os.Exit(0)
			}
			// 搜索Pod名称
			fmt.Println("Searching for pods containing:", input)
			for i, pod := range pods.Items {
				if strings.Contains(pod.Name, input) {
					fmt.Printf("[\u001B[1;31m %d \u001B[0m] %s \u001B[0;32m%s\u001B[0m \n", i, pod.Name, pod.Status.Phase)
				}
			}
		}

		pods, err = clientset.CoreV1().Pods(*namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Printf("Error listing pods: %v\n", err)
			return
		}
	}
}

func handlePodAction(line *liner.State, pod v1.Pod, kubeconfig, namespace *string) {
	fmt.Println("====================================")
	// 高亮显示选中的Pod名称
	fmt.Printf("Selected pod: \033[1;33m %s \033[0m \n", pod.Name)
	fmt.Println("====================================")
	fmt.Println("command action [l, lf, s, q]: ")
	fmt.Println("\u001B[0;31ml\u001B[0m: view all logs")
	fmt.Println("\u001B[0;31mlf\u001B[0m: view rolling logs")
	fmt.Println("\u001B[0;31ms\u001B[0m: enter shell")
	fmt.Println("\u001B[0;31mq\u001B[0m: quit")

	action, _ := line.Prompt("Enter action: ")
	action = strings.TrimSpace(action)

	switch action {
	case "l":
		// 查看日志
		cmd := exec.Command("kubectl", "--kubeconfig", *kubeconfig, "-n", *namespace, "logs", pod.Name)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	case "lf":
		// 查看滚动日志
		cmd := exec.Command("kubectl", "--kubeconfig", *kubeconfig, "-n", *namespace, "logs", "-f", "--tail=1000", pod.Name)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	case "s":
		// 进入shell
		cmd := exec.Command("kubectl", "--kubeconfig", *kubeconfig, "-n", *namespace, "exec", "-it", pod.Name, "--", "/bin/bash")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	case "q":
		fmt.Println("bye bye")
		os.Exit(0)
	default:
		fmt.Println("Invalid action")
	}
}
