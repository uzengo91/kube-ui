package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/peterh/liner"

	"github.com/AlecAivazis/survey/v2"
	v1 "k8s.io/api/core/v1"

	"github.com/olekukonko/tablewriter"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	line       = liner.NewLiner()
	kubeConfig *string
	namespace  *string = new(string)
	k8sClient  *kubernetes.Clientset
)

// 添加新的配置结构体
type KubeConfig struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Namespace string `json:"namespace"`
	Comment   string `json:"comment"`
}

type KubeUIConfig struct {
	Configs []KubeConfig `json:"configs"`
}

func main() {

	defer line.Close()
	line.SetCtrlCAborts(true)

	// 获取配置文件路径

	kubeConfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	namespace = flag.String("namespace", "", "k8s namespace to use")
	flag.Parse()

	// 如果未指定 kubeconfig，尝试读取 ~/.kube-ui
	if *kubeConfig == "" {
		if err := loadKubeUIConfig(); err != nil {
			fmt.Printf("Error loading kube-ui config: %v\n", err)
			return
		}
	}

	// 配置文件不能为空
	if *kubeConfig == "" {
		fmt.Println("Kubeconfig file is required")
		return
	}

	// 使用配置文件创建k8s客户端
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
	if err != nil {
		fmt.Printf("Error building kubeconfig: %v\n", err)
		return
	}

	k8sClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Error creating Kubernetes client: %v\n", err)
		return
	}

	for {
		if *namespace == "" {
			//获取k8s命名空间
			namespaces, err := k8sClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				fmt.Printf("Error listing namespaces: %v\n", err)
				fmt.Printf("获取命名空间列表失败，请检查是否有权限，可手动指定命名空间")
				return
			}
			var namespaceList = make([]string, 0)
			for _, space := range namespaces.Items {
				//fmt.Printf("[\u001B[1;31m %d \u001B[0m] %s\n", i, namespace.Name)
				namespaceList = append(namespaceList, space.Name)
			}
			prompt := &survey.Select{
				Message: "choose k8s namespace:",
				Options: namespaceList,
			}
			err = survey.AskOne(prompt, namespace)
			if err != nil {
				fmt.Printf("Error selecting namespace: %v\n", err)
				return
			}
		}
		var action = new(string)
		prompt := &survey.Select{
			Message: fmt.Sprintf("choose action in namespace %s:", *namespace),
			Options: []string{"pods", "svc", "pvc", "configmap", "exit"},
		}
		err = survey.AskOne(prompt, action)
		if err != nil {
			fmt.Printf("Error selecting action: %v\n", err)
			os.Exit(1)
			return
		}
		switch *action {
		case "pods":
			handleNamespacePodAction()
		case "svc":
			handleNamespaceSvcAction()
		case "configmap":
			handleNamespaceConfigMapAction()
		case "pvc":
			handleNamespacePvcAction()
		case "exit":
			fmt.Println("bye!!!")
			os.Exit(0)
		}

	}

}

// 加载kubeconfig配置
func loadKubeUIConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting home directory: %v", err)
	}

	kubeUIPath := filepath.Join(homeDir, ".kube-ui")
	if _, err := os.Stat(kubeUIPath); err == nil {
		// 读取并解析 .kube-ui 文件
		data, err := os.ReadFile(kubeUIPath)
		if err != nil {
			return fmt.Errorf("error reading .kube-ui file: %v", err)
		}

		var config KubeUIConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("error parsing .kube-ui file: %v", err)
		}

		if len(config.Configs) > 0 {
			// 让用户选择配置
			var configNames []string
			for _, cfg := range config.Configs {
				displayName := cfg.Name
				if cfg.Comment != "" {
					displayName += fmt.Sprintf(" (%s)", cfg.Comment)
				}
				configNames = append(configNames, displayName)
			}
			configNames = append(configNames, "exit")

			var selectedIndex int
			prompt := &survey.Select{
				Message: "Choose kubernetes config:",
				Options: configNames,
			}
			survey.AskOne(prompt, &selectedIndex)
			if selectedIndex == len(configNames)-1 {
				fmt.Println("bye!!!")
				os.Exit(0)
			}

			selectedConfig := config.Configs[selectedIndex]
			*kubeConfig = selectedConfig.Path
			if selectedConfig.Namespace != "" {
				*namespace = selectedConfig.Namespace
			}
		}
	}
	return nil
}

func handleNamespacePvcAction() {
	// 获取Pvc列表
	pvcList, err := k8sClient.CoreV1().PersistentVolumeClaims(*namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error listing pvcs: %v\n", err)
		fmt.Printf("获取命名空间%s下的Pvc列表失败", *namespace)
		return
	}
	// 打印Pvc列表
	fmt.Println("pvc in namespace", *namespace)
	printPvcTable(pvcList, "", nil)
	for {
		input := ""
		prompt := &survey.Input{
			Message: "Enter pvc number or search, exit to quit: ",
		}
		survey.AskOne(prompt, &input)

		// 	// 检查输入是否为数字
		pvcNumber, err := strconv.Atoi(input)
		if err == nil && pvcNumber >= 0 && pvcNumber < len(pvcList.Items) {
			selectedPvc := pvcList.Items[pvcNumber]
			handlePvcAction(line, selectedPvc)
			pvcList, _ = k8sClient.CoreV1().PersistentVolumeClaims(*namespace).List(context.TODO(), metav1.ListOptions{})
			printPvcTable(pvcList, "", nil)
		} else {
			//如果== exit 退出
			if input == "exit" {
				return
			}
			printPvcTable(pvcList, input, func(pvc v1.PersistentVolumeClaim, input string) bool {
				return strings.Contains(pvc.Name, input)
			})
		}
	}
}

func handlePvcAction(line *liner.State, selectedPvc v1.PersistentVolumeClaim) {
	// 选中的Pvc
	for {
		fmt.Println("====================================")
		// 高亮显示选中的Pvc名称
		fmt.Printf("Selected Pvc: \033[1;33m %s \033[0m \n", selectedPvc.Name)
		fmt.Println("====================================")
		fmt.Println("command action [p, exit]: ")
		fmt.Println("\u001B[0;31m p \u001B[0m: print Pvc info")
		fmt.Println("\u001B[0;31m exit \u001B[0m: quit current action")

		action, _ := line.Prompt("Enter action: ")
		action = strings.TrimSpace(action)

		switch action {
		case "p":
			execCommand("get", "pvc", selectedPvc.Name, "-o", "yaml")
		case "exit":
			return
		default:
			fmt.Println("Invalid action")
		}
	}
}

func handleNamespaceConfigMapAction() {
	// 获取ConfigMap列表
	configMaps, err := k8sClient.CoreV1().ConfigMaps(*namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error listing configmaps: %v\n", err)
		fmt.Printf("获取命名空间%s下的ConfigMap列表失败", *namespace)
		return
	}
	// 打印ConfigMap列表
	fmt.Println("ConfigMaps in namespace", *namespace)
	printConfigMapTable(configMaps, "", nil)

	for {
		input := ""
		prompt := &survey.Input{
			Message: "Enter pod number or search, exit to quit: ",
		}
		survey.AskOne(prompt, &input)

		// 	// 检查输入是否为数字
		podNumber, err := strconv.Atoi(input)
		if err == nil && podNumber >= 0 && podNumber < len(configMaps.Items) {
			selectedConfigMap := configMaps.Items[podNumber]
			handleConfigMapAction(line, selectedConfigMap)
			configMaps, _ = k8sClient.CoreV1().ConfigMaps(*namespace).List(context.TODO(), metav1.ListOptions{})
			printConfigMapTable(configMaps, "", nil)
		} else {
			//如果== exit 退出
			if input == "exit" {
				return
			}
			printConfigMapTable(configMaps, input, func(pod v1.ConfigMap, input string) bool {
				return strings.Contains(pod.Name, input)
			})
		}
	}
}

func handleConfigMapAction(line *liner.State, selectedConfigMap v1.ConfigMap) {
	for {
		fmt.Println("====================================")
		// 高亮显示选中的ConfigMap名称
		fmt.Printf("Selected ConfigMap: \033[1;33m %s \033[0m \n", selectedConfigMap.Name)
		fmt.Println("====================================")
		fmt.Println("command action [p, exit]: ")
		fmt.Println("\u001B[0;31m p \u001B[0m: print ConfigMap info")
		fmt.Println("\u001B[0;31m exit \u001B[0m: quit current action")

		action, _ := line.Prompt("Enter action: ")
		action = strings.TrimSpace(action)

		switch action {
		case "p":
			execCommand("get", "configmap", selectedConfigMap.Name, "-o", "yaml")
		case "exit":
			return
		default:
			fmt.Println("Invalid action")
		}
	}
}

func handleNamespaceSvcAction() {
	// 获取Service列表
	svcList, err := k8sClient.CoreV1().Services(*namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error listing services: %v\n", err)
		fmt.Printf("获取命名空间%s下的Service列表失败", *namespace)
		return
	}
	// 打印Service列表
	fmt.Println("Services in namespace", *namespace)
	printSvcTable(svcList, "", nil)
	for {
		input := ""
		prompt := &survey.Input{
			Message: "Enter pod number or search, exit to quit: ",
		}
		survey.AskOne(prompt, &input)

		// 	// 检查输入是否为数字
		podNumber, err := strconv.Atoi(input)
		if err == nil && podNumber >= 0 && podNumber < len(svcList.Items) {
			selectedSvc := svcList.Items[podNumber]
			handleSvcAction(line, selectedSvc)
			svcList, _ = k8sClient.CoreV1().Services(*namespace).List(context.TODO(), metav1.ListOptions{})
			printSvcTable(svcList, "", nil)
		} else {
			//如果== exit 退出
			if input == "exit" {
				return
			}
			printSvcTable(svcList, input, func(pod v1.Service, input string) bool {
				return strings.Contains(pod.Name, input)
			})
		}
	}

}

func handleNamespacePodAction() {

	// 获取Pod列表
	pods, err := k8sClient.CoreV1().Pods(*namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error listing pods: %v\n", err)
		fmt.Printf("获取命名空间%s下的Pod列表失败", *namespace)
		return
	}

	// 显示Pod列表并标号
	fmt.Println("Pods in namespace", *namespace)
	// for i, pod := range pods.Items {
	// 	fmt.Printf("[\u001B[1;31m %d \u001B[0m] %s \u001B[0;32m%s\u001B[0m \n", i, pod.Name, pod.Status.Phase)
	// }
	printPodTable(pods, "", nil)
	for {

		input := ""
		prompt := &survey.Input{
			Message: "Enter pod number or search, exit to quit: ",
		}
		survey.AskOne(prompt, &input)

		// 	// 检查输入是否为数字
		podNumber, err := strconv.Atoi(input)
		if err == nil && podNumber >= 0 && podNumber < len(pods.Items) {
			selectedPod := pods.Items[podNumber]
			handlePodAction(line, selectedPod)
			pods, _ = k8sClient.CoreV1().Pods(*namespace).List(context.TODO(), metav1.ListOptions{})
			printPodTable(pods, "", nil)
		} else {
			//如果== exit 退出
			if input == "exit" {
				return
			}
			// 搜索Pod名称
			//fmt.Println("Searching for pods containing:", input)
			// for i, pod := range pods.Items {
			// 	if strings.Contains(pod.Name, input) {
			// 		fmt.Printf("[\u001B[1;31m %d \u001B[0m] %s \u001B[0;32m%s\u001B[0m \n", i, pod.Name, pod.Status.Phase)
			// 	}
			// }
			printPodTable(pods, input, func(pod v1.Pod, input string) bool {
				return strings.Contains(pod.Name, input)
			})
		}
	}

}

func printPvcTable(pvcList *v1.PersistentVolumeClaimList, s string, f func(pvc v1.PersistentVolumeClaim, input string) bool) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Number", "Name", "Status", "StorageClass", "Capacity", "AccessMode"})
	for i, pvc := range pvcList.Items {
		if f != nil && !f(pvc, s) {
			continue
		}
		var models []string = make([]string, 0)
		for _, model := range pvc.Spec.AccessModes {
			models = append(models, string(model))
		}
		table.Append([]string{
			fmt.Sprintf("%d", i),
			pvc.Name,
			string(pvc.Status.Phase),
			*pvc.Spec.StorageClassName,
			pvc.Status.Capacity.Storage().String(),
			strings.Join(models, ","),
		})
	}
	table.Render()

}

func printConfigMapTable(configMapList *v1.ConfigMapList, input string, f func(pod v1.ConfigMap, input string) bool) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Number", "Name", "Data"})
	for i, pod := range configMapList.Items {
		if f != nil && !f(pod, input) {
			continue
		}
		table.Append([]string{fmt.Sprintf("%d", i), pod.Name, fmt.Sprintf("%d", len(pod.Data))})
	}
	table.Render()
}

func printPodTable(pods *v1.PodList, input string, f func(pod v1.Pod, input string) bool) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Number", "pod-Name", "pod-Status", "restart-times", "age"})
	for i, pod := range pods.Items {
		if f != nil && !f(pod, input) {
			continue
		}
		age := metav1.Now().Sub(pod.Status.StartTime.Time).Round(time.Minute)
		restartCount := 0
		table.Append([]string{
			fmt.Sprintf("%d", i),
			pod.Name,
			string(pod.Status.Phase),
			fmt.Sprintf("%d", restartCount),
			age.String(),
		})
	}
	table.Render()
}

func printSvcTable(pods *v1.ServiceList, input string, f func(pod v1.Service, input string) bool) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Number", "Name", "TYPE", "CLUSTER-IP", "EXTERNAL-IP", "PORT(S)"})
	for i, pod := range pods.Items {
		if f != nil && !f(pod, input) {
			continue
		}
		var ports []string = make([]string, 0)
		for _, port := range pod.Spec.Ports {
			ports = append(ports, fmt.Sprintf("%d/%s", port.Port, port.Protocol))
		}
		table.Append([]string{
			fmt.Sprintf("%d", i),
			pod.Name,
			string(pod.Spec.Type),
			pod.Spec.ClusterIP,
			strings.Join(pod.Spec.ExternalIPs, ","),
			strings.Join(ports, ","),
		})
	}
	table.Render()
}

func handleSvcAction(line *liner.State, svc v1.Service) {
	for {
		fmt.Println("====================================")
		// 高亮显示选中的svc名称
		fmt.Printf("Selected svc: \033[1;33m %s \033[0m \n", svc.Name)
		fmt.Println("====================================")
		fmt.Println("command action [p fw exit]: ")
		fmt.Println("\u001B[0;31m p \u001B[0m: print svc info")
		fmt.Println("\u001B[0;31m fw \u001B[0m: forward svc port")
		fmt.Println("\u001B[0;31m exit \u001B[0m: quit current action")

		action, _ := line.Prompt("Enter action: ")
		action = strings.TrimSpace(action)

		switch action {
		case "p":
			execCommand("get", "svc", svc.Name, "-o", "yaml")
		case "fw":
			ports, _ := line.Prompt("please enter forward ports, example: \"localPort1:svcPort1 localPort2:svcPort2\", so you can input \"8080:80 9090:90\" ")
			ports = strings.TrimSpace(ports)
			forwardPort := []string{"port-forward", "svc/" + svc.Name}
			for _, portValue := range strings.Split(ports, " ") {
				portPairNew := strings.TrimSpace(portValue)
				if portPairNew == "" {
					continue
				}
				forwardPort = append(forwardPort, portPairNew)
			}
			execCommand(forwardPort...)
		case "exit":
			return
		default:
			fmt.Println("Invalid action")
		}
	}
}

func handlePodAction(line *liner.State, pod v1.Pod) {
	for {
		fmt.Println("====================================")
		// 高亮显示选中的Pod名称
		fmt.Printf("Selected pod: \033[1;33m %s \033[0m \n", pod.Name)
		fmt.Println("====================================")
		fmt.Println("command action [p, l, lf, s, e, fw, cp, u, exit]: ")
		fmt.Println("\u001B[0;31m p \u001B[0m: print pod info")
		fmt.Println("\u001B[0;31m l \u001B[0m: view all logs")
		fmt.Println("\u001B[0;31m lf \u001B[0m: view rolling logs")
		fmt.Println("\u001B[0;31m s \u001B[0m: enter shell")
		fmt.Println("\u001B[0;31m e \u001B[0m: view pod events")
		fmt.Println("\u001B[0;31m fw \u001B[0m: port forward remote port to local")
		fmt.Println("\u001B[0;31m cp \u001B[0m: copy remote file to current path, download file name is remote file name")
		fmt.Println("\u001B[0;31m u \u001B[0m: upload local file to remote pod")
		fmt.Println("\u001B[0;31m exit \u001B[0m: quit current action")

		action, _ := line.Prompt("Enter action: ")
		action = strings.TrimSpace(action)

		switch action {
		case "p":
			// 查看pod信息
			execCommand("get", "pod", pod.Name, "-o", "yaml")
			// cmd := exec.Command("kubectl", "--kubeconfig", *kubeConfig, "-n", *namespace, "get", "pod", pod.Name, "-o", "yaml")
			// cmd.Stdout = os.Stdout
			// cmd.Stderr = os.Stderr
			// cmd.Run()
		case "l":
			// 查看日志
			execCommand("logs", pod.Name)
		case "lf":
			// 查看滚动日志
			execCommand("logs", "-f", "--tail=1000", pod.Name)
		case "cp":
			// 复制文件
			src, _ := line.Prompt("Enter remote file path: ")
			// 默认使用远程文件名
			dst := src[strings.LastIndex(src, "/")+1:]
			execCommand("cp", pod.Name+":"+src, dst)
		case "u":
			// 上传文件
			src, _ := line.Prompt("Enter local file path: ")
			dst, _ := line.Prompt("Enter remote file path: ")
			execCommand("cp", src, pod.Name+":"+dst)
		case "s":
			// 处理容器选择
			if len(pod.Spec.Containers) == 1 {
				// 只有一个容器时直接进入
				execCommand("exec", "-it", pod.Name, "--", "/bin/bash")
			} else {
				// 多个容器时显示选择表格
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"Number", "Container Name"})

				for i, container := range pod.Spec.Containers {
					table.Append([]string{
						fmt.Sprintf("%d", i),
						container.Name,
					})
				}
				table.Render()

				// 让用户选择容器
				var containerNum string
				prompt := &survey.Input{
					Message: "Enter container number to exec into: ",
				}
				survey.AskOne(prompt, &containerNum)

				if num, err := strconv.Atoi(containerNum); err == nil && num >= 0 && num < len(pod.Spec.Containers) {
					execCommand("exec", "-it", pod.Name, "-c", pod.Spec.Containers[num].Name, "--", "/bin/bash")
				} else {
					fmt.Println("Invalid container number")
				}
			}
		case "e":
			// 查看pod事件
			printPodEvents(pod)
		case "fw":
			// 端口转发
			ports, _ := line.Prompt("please enter forward ports, example: \"localPort1:podPort1 localPort2:podPort2\", so you can input \"8080:80 9090:90\" ")
			ports = strings.TrimSpace(ports)
			forwardPort := []string{"port-forward", "pod/" + pod.Name}
			for _, portPair := range strings.Split(ports, " ") {
				portPairNew := strings.TrimSpace(portPair)
				if portPairNew == "" {
					continue
				}
				forwardPort = append(forwardPort, portPairNew)
			}
			execCommand(forwardPort...)
		case "exit":
			return
		default:
			fmt.Println("Invalid action")
		}
	}
}

// 添加新函数用于打印pod事件
func printPodEvents(pod v1.Pod) {
	// 获取pod相关的事件
	events, err := k8sClient.CoreV1().Events(*namespace).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Pod", pod.Name),
	})
	if err != nil {
		fmt.Printf("Error getting pod events: %v\n", err)
		return
	}

	// 创建表格
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Type", "Reason", "Age", "From", "Message"})

	// 添加事件数据
	for _, event := range events.Items {
		age := metav1.Now().Sub(event.FirstTimestamp.Time).Round(time.Minute)
		table.Append([]string{
			event.Type,
			event.Reason,
			age.String(),
			event.Source.Component,
			event.Message,
		})
	}

	// 渲染表格
	table.Render()
}

func execCommand(arg ...string) {
	defaultArg := []string{"--kubeconfig", *kubeConfig, "-n", *namespace}
	arg = append(defaultArg, arg...)
	cmd := exec.Command("kubectl", arg...)
	fmt.Println("exec command: \u001B[0;31m " + cmd.String() + " \u001B[0m")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}
