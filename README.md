## 项目简介

交互式连接k8s, 快捷进行操作

## 遇到问题

- 动态库版本过低

```bash
./kube-ui: /lib/x86_64-linux-gnu/libc.so.6: version `GLIBC_2.34' not found (required by ./kube-ui)
./kube-ui: /lib/x86_64-linux-gnu/libc.so.6: version `GLIBC_2.32' not found (required by ./kube-ui)
```

查看依赖的动态库
`ldd ./bin/kube-ui`

```bash
➜  kube-ui git:(master) ldd ./bin/kube-ui
        linux-vdso.so.1 (0x00007fff44cfa000)
        libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6 (0x00007fcbf9af2000)
        /lib64/ld-linux-x86-64.so.2 (0x00007fcbf9d27000)
```

解决办法
构建过程中 指定 `CGO_ENABLED=0`

再次查看依赖的动态库

```bash
➜  kube-ui git:(master) ✗ ldd ./bin/kube-ui
        not a dynamic executable
```
