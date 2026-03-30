# 从命令行轻松运行 Jenkins 任务

<meta name="google-site-verification" content="Wl2WZRolJ6omFNTQRguTy0GRQU41taSDq20n4Qgz05c" />

该工具可以从命令行/终端启动 Jenkins 构建/任务。
执行效果如下：

![终端演示](assets/demo.gif)

## 安装

从 [最新发布版本](https://github.com/gocruncher/jenkins-job-cli/releases) 获取适合您平台的版本：

#### Linux

```bash
sudo wget https://github.com/gocruncher/jenkins-job-cli/releases/download/v1.1.2/jenkins-job-cli-1.1.2-linux-amd64 -O /usr/local/bin/jj
sudo chmod +x /usr/local/bin/jj
```

#### OS X brew

```bash
# brew tap gocruncher/tap
# brew install jj
brew install jeffzhangc/tap/jenkins-job-cli
```

#### OS X bash

```bash
sudo curl -Lo /usr/local/bin/jj https://github.com/gocruncher/jenkins-job-cli/releases/download/v1.1.2/jenkins-job-cli-1.1.2-darwin-amd64
sudo chmod +x /usr/local/bin/jj
```

## 快速开始

### 配置多个 Jenkins 访问

```bash
jj set dev_jenkins --url "https://myjenkins.com" --login admin --token 11aa0926784999dab5
```

其中 token 可在您的 Jenkins 个人配置页面获取。访问 Jenkins Web 界面，点击页面右上角的您的姓名，然后点击"Configure"即可查看您的 API token。

如果 Jenkins 无需授权即可访问：

```bash
jj set dev_jenkins --url "https://myjenkins.com"
```

或者以交互模式运行以下命令：

```bash
jj set dev_jenkins
```

### Shell 自动补全

建议启用 shell 自动补全以便于使用。运行以下命令：

```bash
# zsh 补全：
echo 'source <(jj completion zsh)' >>~/.zshrc

# bash 补全：
echo 'source <(jj completion bash)' >>~/.bashrc
```

如果由于某些原因无法工作，可以尝试以下命令来排查问题：

```bash
jj completion check
```

### 使用示例

```bash
# 配置 Jenkins 访问
jj set dev-jenkins

# 在当前 Jenkins 中启动 'app-build' 任务
jj run app-build

# 在名为 prod 的 Jenkins 中启动 'web-build' 任务
jj run -n prod web-build

# 将特定 Jenkins 设置为默认
jj use PROD

# 列出所有正在运行的任务
jj console

# 显示最新任务的控制台输出
jj console app-xxx
```

## 历史记录管理

`jj history` 命令允许您保存、管理和快速重新运行常用的 Jenkins 任务命令，并使用自定义别名。此功能可帮助您避免重复输入长命令。

### 保存快速命令

运行任务后，系统会提示您使用别名保存该命令。您也可以通过手动方式添加命令：

```bash
# 手动添加命令并设置别名（交互式）
jj history add "jj run cc-stat3 -n aicc -a env=dev-93 -a DeliveryModel=BuildAndDeploy"

# 命令将解析并显示：
# - 任务名称：cc-stat3
# - 环境：aicc
# - 参数：env=dev-93, DeliveryModel=BuildAndDeploy 等
# 然后会提示您输入别名
```

**别名重复处理：**
- 如果别名已存在，会询问是否覆盖
- 输入 `y` 或 `yes` 覆盖现有命令
- 输入 `n` 或 `N` 保留现有命令并输入新别名

### 列出已保存的命令

```bash
# 列出所有已保存的快速命令
jj history list

# 列出特定环境中的命令
jj history list -e prod

# 限制结果数量
jj history list -l 10

# 显示所有详细信息，包括完整命令
jj history list -a

# 以不同格式输出
jj history list -f json
jj history list -f yaml
```

### 运行已保存的命令

```bash
# 通过别名运行已保存的命令
jj history run myjob_quick

# 同时运行多个命令
jj history run alias1 alias2 alias3

# 强制运行，无需确认
jj history run -f myjob_quick
```

### 查看命令详情

```bash
# 查看已保存命令的详细信息
jj history view alias1

# 查看多个命令
jj history view alias1 alias2

# 以 JSON 格式输出
jj history view -f json alias1
```

### 搜索命令

```bash
# 通过关键词搜索命令
jj history search "prod"

# 在特定环境中搜索
jj history search -e dev "deploy"

# 以 JSON 格式输出
jj history search -f json "job"
```

### 删除命令

```bash
# 删除已保存的命令
jj history delete alias1

# 删除多个命令
jj history delete alias1 alias2

# 强制删除，无需确认
jj history delete -f alias1
```

### 清空所有历史记录

```bash
# 清空所有已保存的命令（需要确认）
jj history clear

# 强制清空，无需确认
jj history clear -f
```

### 导出和导入

```bash
# 将历史记录导出到文件
jj history export history.json
jj history export history.yaml

# 导出特定环境
jj history export -e prod prod.yaml

# 从文件导入历史记录
jj history import backup.yaml

# 导入时合并而不是替换
jj history import -m backup.yaml
```

### 历史命令别名

- `history` / `hist` / `h` - 主历史命令
- `list` / `ls` / `l` - 列出命令
- `run` / `r` / `exec` - 运行命令
- `view` / `v` / `show` / `info` - 查看详情
- `delete` / `del` / `rm` / `remove` - 删除命令
- `clear` / `clean` / `clr` - 清空所有
- `search` / `s` / `find` / `grep` - 搜索命令
- `export` / `exp` / `save` - 导出历史
- `import` / `imp` / `load` - 导入历史
- `add` / `a` / `save` - 添加命令到历史

## 功能特性

- ✅ 从命令行运行 Jenkins 任务
- ✅ 管理多个 Jenkins 实例
- ✅ 使用别名保存和重新运行常用命令
- ✅ 搜索和管理命令历史记录
- ✅ 导出/导入命令历史记录
- ✅ 任务取消（Ctrl+C 键）
- ✅ 调整输出大小（按回车键）
- ✅ 子任务输出
- ✅ 显示控制台信息
- ✅ Shell 自动补全支持

## 有用的包

- [cobra](https://github.com/spf13/cobra) - 用于创建强大的现代 CLI 的库
- [chalk](https://github.com/chalk/chalk) – 终端字符串样式处理
- [bar](https://github.com/superhawk610/bar) - 灵活的 ASCII 进度条

## 待办事项

- 添加通过登录/密码和 RSA 密钥进行授权
- 支持终端窗口大小调整

## 类似项目

- [jcli](https://github.com/jenkins-zh/jenkins-cli/) 使用 Golang 编写，可以管理多个 Jenkins
- [jenni](https://github.com/m-sureshraj/jenni)

## 许可证

`jenkins-job-cli` 是在 [MIT](LICENSE) 许可证下发布的开源软件。
