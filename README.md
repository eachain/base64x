# base64x

base64x 是一个类似 "base64" 的命令行工具。它主要解决多段base64编码解析，以及实时输入输出的问题。

## 多段编码

比如：

```bash
$ echo -n a | base64
YQ==
```

解码多段时：

```bash
$ echo YQ==YQ== | base64 -d

```

base64 命令无法正常解析。用 base64x 解码：

```bash
$ echo YQ==YQ== | base64x -d
aa
```

## 实时输入输出

```bash
$ base64
abc
def
<--- ctrl+D (EOF) here
YWJjCmRlZgo=
```

```base
$ base64x
abc
YWJjCg==
def
ZGVmCg==
```

base64 在读到EOF或足够多（看base64具体实现方式）数据后，才会输出编解码内容。 base64x 则是读到多少编解码多少，实时输出，这在要求实时输入输出时将非常有用。
