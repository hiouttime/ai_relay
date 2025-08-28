<template>
  <div class="help-container">
    <!-- 系统选择标签 -->
    <div class="system-tabs">
      <div class="tabs-wrapper">
        <t-button
          v-for="system in supportedSystems"
          :key="system.key"
          :variant="activeTutorialSystem === system.key ? 'base' : 'outline'"
          :theme="activeTutorialSystem === system.key ? 'primary' : 'default'"
          class="system-tab-btn"
          @click="activeTutorialSystem = system.key"
        >
          <t-icon style="margin-top: 2px" :name="system.icon" />
          <span style="margin-left: 4px">{{ system.name }}</span>
        </t-button>
      </div>
    </div>

    <!-- Windows 教程 -->
    <div v-if="activeTutorialSystem === 'windows'" class="tutorial-content">
      <!-- 第一步：安装 Node.js -->
      <div class="tutorial-step">
        <h3 class="step-title">
          <span class="step-number">1</span>
          安装 Node.js 环境
        </h3>
        <p class="step-description">Claude Code 需要 Node.js 环境才能运行。</p>

        <t-card class="step-card">
          <div class="step-content">
            <h4 class="method-title">
              <t-icon name="logo-windows" />
              Windows 安装方法
            </h4>

            <div class="method-section">
              <h5>方法一：官网下载（推荐）</h5>
              <ol class="install-steps">
                <li>打开浏览器访问 <code>https://nodejs.org/</code></li>
                <li>点击 "LTS" 版本进行下载（推荐长期支持版本）</li>
                <li>下载完成后双击 <code>.msi</code> 文件</li>
                <li>按照安装向导完成安装，保持默认设置即可</li>
              </ol>
            </div>

            <div class="method-section">
              <h5>方法二：使用包管理器</h5>
              <p>如果你安装了 Chocolatey 或 Scoop，可以使用命令行安装：</p>
              <div class="code-block">
                <pre><code># 使用 Chocolatey
choco install nodejs

# 或使用 Scoop
scoop install nodejs</code></pre>
              </div>
            </div>

            <t-alert theme="info" class="note-alert">
              <h6>Windows 注意事项</h6>
              <ul>
                <li>建议使用 PowerShell 而不是 CMD</li>
                <li>如果遇到权限问题，尝试以管理员身份运行</li>
                <li>某些杀毒软件可能会误报，需要添加白名单</li>
              </ul>
            </t-alert>
          </div>
        </t-card>

        <!-- 验证安装 -->
        <t-card class="verify-card">
          <h4 class="verify-title">验证安装是否成功</h4>
          <p>安装完成后，打开 PowerShell 或 CMD，输入以下命令：</p>
          <div class="code-block">
            <pre><code>node --version
npm --version</code></pre>
          </div>
          <p class="verify-note">如果显示版本号，说明安装成功了！</p>
        </t-card>
      </div>

      <!-- 第二步：安装 Git Bash -->
      <div class="tutorial-step">
        <h3 class="step-title">
          <span class="step-number">2</span>
          安装 Git Bash
        </h3>
        <p class="step-description">
          Windows 环境下需要使用 Git Bash 安装Claude code。安装完成后，环境变量设置和使用 Claude Code 仍然在普通的
          PowerShell 或 CMD 中进行。
        </p>

        <t-card class="step-card">
          <div class="step-content">
            <h4 class="method-title">
              <t-icon name="logo-github" />
              下载并安装 Git for Windows
            </h4>
            <ol class="install-steps">
              <li>访问 <code>https://git-scm.com/downloads/win</code></li>
              <li>点击 "Download for Windows" 下载安装包</li>
              <li>运行下载的 <code>.exe</code> 安装文件</li>
              <li>在安装过程中保持默认设置，直接点击 "Next" 完成安装</li>
            </ol>

            <t-alert theme="success" class="note-alert">
              <h6>安装完成后</h6>
              <ul>
                <li>在任意文件夹右键可以看到 "Git Bash Here" 选项</li>
                <li>也可以从开始菜单启动 "Git Bash"</li>
                <li>只需要在 Git Bash 中运行 npm install 命令</li>
                <li>后续的环境变量设置和使用都在 PowerShell/CMD 中</li>
              </ul>
            </t-alert>
          </div>
        </t-card>

        <!-- 验证安装 -->
        <t-card class="verify-card">
          <h4 class="verify-title">验证 Git Bash 安装</h4>
          <p>打开 Git Bash，输入以下命令验证：</p>
          <div class="code-block">
            <pre><code>git --version</code></pre>
          </div>
          <p class="verify-note">如果显示 Git 版本号，说明安装成功！</p>
        </t-card>
      </div>

      <!-- 第三步：安装 Claude Code -->
      <div class="tutorial-step">
        <h3 class="step-title">
          <span class="step-number">3</span>
          安装 Claude Code
        </h3>

        <t-card class="step-card">
          <div class="step-content">
            <h4 class="method-title">
              <t-icon name="download" />
              安装 Claude Code
            </h4>
            <p>打开 Git Bash（重要：不要使用 PowerShell），运行以下命令：</p>
            <div class="code-block">
              <pre><code># 在 Git Bash 中全局安装 Claude Code
npm install -g @anthropic-ai/claude-code</code></pre>
            </div>
            <p>这个命令会从 npm 官方仓库下载并安装最新版本的 Claude Code。</p>

            <t-alert theme="warning" class="note-alert">
              <h6>重要提醒</h6>
              <ul>
                <li>必须在 Git Bash 中运行，不要在 PowerShell 中运行</li>
                <li>如果遇到权限问题，可以尝试在 Git Bash 中使用 sudo 命令</li>
              </ul>
            </t-alert>
          </div>
        </t-card>

        <!-- 验证安装 -->
        <t-card class="verify-card">
          <h4 class="verify-title">验证 Claude Code 安装</h4>
          <p>安装完成后，输入以下命令检查是否安装成功：</p>
          <div class="code-block">
            <pre><code>claude --version</code></pre>
          </div>
          <p class="verify-note">如果显示版本号，恭喜你！Claude Code 已经成功安装了。</p>
        </t-card>
      </div>

      <!-- 第四步：设置环境变量 -->
      <div class="tutorial-step">
        <h3 class="step-title">
          <span class="step-number">4</span>
          设置环境变量
        </h3>

        <t-card class="step-card">
          <div class="step-content">
            <h4 class="method-title">
              <t-icon name="setting" />
              配置 Claude Code 环境变量
            </h4>
            <p>为了让 Claude Code 连接到你的中转服务，需要设置两个环境变量：</p>

            <div class="method-section">
              <h5>方法一：PowerShell 临时设置（推荐）</h5>
              <p>在 PowerShell 中运行以下命令：</p>
              <div class="code-block">
                <pre><code>$env:ANTHROPIC_BASE_URL = "{{ currentBaseUrl }}"
$env:ANTHROPIC_AUTH_TOKEN = "你的API密钥"</code></pre>
              </div>
              <p class="note-text">💡 记得将 "你的API密钥" 替换为在 "API Keys" 标签页中创建的实际密钥。</p>
            </div>

            <div class="method-section">
              <h5>方法二：系统环境变量（永久设置）</h5>
              <ol class="install-steps">
                <li>右键"此电脑" → "属性" → "高级系统设置"</li>
                <li>点击"环境变量"按钮</li>
                <li>在"用户变量"或"系统变量"中点击"新建"</li>
                <li>添加以下两个变量：</li>
              </ol>
              <div class="env-vars">
                <div class="env-var">
                  <strong>变量名：</strong> ANTHROPIC_BASE_URL<br />
                  <strong>变量值：</strong> <code>{{ currentBaseUrl }}</code>
                </div>
                <div class="env-var">
                  <strong>变量名：</strong> ANTHROPIC_AUTH_TOKEN<br />
                  <strong>变量值：</strong> <code>你的API密钥</code>
                </div>
              </div>
            </div>
          </div>
        </t-card>

        <!-- 验证环境变量设置 -->
        <t-card class="verify-card">
          <h4 class="verify-title">验证环境变量设置</h4>
          <p>设置完环境变量后，可以通过以下命令验证是否设置成功：</p>

          <div class="verify-section">
            <h5>在 PowerShell 中验证：</h5>
            <div class="code-block">
              <pre><code>echo $env:ANTHROPIC_BASE_URL
echo $env:ANTHROPIC_AUTH_TOKEN</code></pre>
            </div>
          </div>

          <div class="verify-section">
            <h5>在 CMD 中验证：</h5>
            <div class="code-block">
              <pre><code>echo %ANTHROPIC_BASE_URL%
echo %ANTHROPIC_AUTH_TOKEN%</code></pre>
            </div>
          </div>

          <div class="expected-output">
            <p><strong>预期输出示例：</strong></p>
            <div class="output-example">
              <div>{{ currentBaseUrl }}</div>
              <div>cr_xxxxxxxxxxxxxxxxxx</div>
            </div>
            <p class="verify-note">💡 如果输出为空或显示变量名本身，说明环境变量设置失败，请重新设置。</p>
          </div>
        </t-card>
      </div>

      <!-- 第五步：开始使用 -->
      <div class="tutorial-step">
        <h3 class="step-title">
          <span class="step-number">5</span>
          开始使用 Claude Code
        </h3>

        <t-card class="step-card">
          <div class="step-content">
            <p>现在你可以开始使用 Claude Code 了！</p>

            <div class="usage-section">
              <h5>启动 Claude Code</h5>
              <div class="code-block">
                <pre><code>claude</code></pre>
              </div>
            </div>

            <div class="usage-section">
              <h5>在特定项目中使用</h5>
              <div class="code-block">
                <pre><code># 进入你的项目目录
cd C:\path\to\your\project

# 启动 Claude Code
claude</code></pre>
              </div>
            </div>
          </div>
        </t-card>
      </div>

      <!-- Windows 故障排除 -->
      <div class="tutorial-step">
        <h3 class="step-title">
          <t-icon name="tools" />
          Windows 常见问题解决
        </h3>

        <div class="troubleshooting">
          <t-collapse>
            <t-collapse-panel header="安装时提示 'permission denied' 错误">
              <p>这通常是权限问题，尝试以下解决方法：</p>
              <ul>
                <li>以管理员身份运行 PowerShell</li>
                <li>或者配置 npm 使用用户目录：<code>npm config set prefix %APPDATA%\npm</code></li>
              </ul>
            </t-collapse-panel>

            <t-collapse-panel header="PowerShell 执行策略错误">
              <p>如果遇到执行策略限制，运行：</p>
              <div class="code-block">
                <pre><code>Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser</code></pre>
              </div>
            </t-collapse-panel>

            <t-collapse-panel header="环境变量设置后不生效">
              <p>设置永久环境变量后需要：</p>
              <ul>
                <li>重新启动 PowerShell 或 CMD</li>
                <li>或者注销并重新登录 Windows</li>
                <li>验证设置：<code>echo $env:ANTHROPIC_BASE_URL</code></li>
              </ul>
            </t-collapse-panel>
          </t-collapse>
        </div>
      </div>
    </div>

    <!-- macOS 教程 -->
    <div v-else-if="activeTutorialSystem === 'macos'" class="tutorial-content">
      <!-- macOS 安装步骤基本相同，只是命令略有不同 -->
      <div class="tutorial-step">
        <h3 class="step-title">
          <span class="step-number">1</span>
          安装 Node.js 环境
        </h3>
        <p class="step-description">Claude Code 需要 Node.js 环境才能运行。</p>

        <t-card class="step-card">
          <div class="step-content">
            <h4 class="method-title">
              <t-icon name="logo-apple" />
              macOS 安装方法
            </h4>

            <div class="method-section">
              <h5>方法一：使用 Homebrew（推荐）</h5>
              <p>如果你已经安装了 Homebrew，使用它安装 Node.js 会更方便：</p>
              <div class="code-block">
                <pre><code># 更新 Homebrew
brew update

# 安装 Node.js
brew install node</code></pre>
              </div>
            </div>

            <div class="method-section">
              <h5>方法二：官网下载</h5>
              <ol class="install-steps">
                <li>访问 <code>https://nodejs.org/</code></li>
                <li>下载适合 macOS 的 LTS 版本</li>
                <li>打开下载的 <code>.pkg</code> 文件</li>
                <li>按照安装程序指引完成安装</li>
              </ol>
            </div>

            <t-alert theme="info" class="note-alert">
              <h6>macOS 注意事项</h6>
              <ul>
                <li>如果遇到权限问题，可能需要使用 <code>sudo</code></li>
                <li>首次运行可能需要在系统偏好设置中允许</li>
                <li>建议使用 Terminal 或 iTerm2</li>
              </ul>
            </t-alert>
          </div>
        </t-card>
      </div>

      <!-- macOS 其他步骤类似，环境变量设置使用 export 命令 -->
      <div class="tutorial-step">
        <h3 class="step-title">
          <span class="step-number">2</span>
          安装 Claude Code
        </h3>

        <t-card class="step-card">
          <div class="step-content">
            <h4 class="method-title">
              <t-icon name="download" />
              安装 Claude Code
            </h4>
            <p>打开 Terminal，运行以下命令：</p>
            <div class="code-block">
              <pre><code># 全局安装 Claude Code
npm install -g @anthropic-ai/claude-code</code></pre>
            </div>
            <p>如果遇到权限问题，可以使用 sudo：</p>
            <div class="code-block">
              <pre><code>sudo npm install -g @anthropic-ai/claude-code</code></pre>
            </div>
          </div>
        </t-card>
      </div>

      <div class="tutorial-step">
        <h3 class="step-title">
          <span class="step-number">3</span>
          设置环境变量
        </h3>

        <t-card class="step-card">
          <div class="step-content">
            <h4 class="method-title">
              <t-icon name="setting" />
              配置 Claude Code 环境变量
            </h4>

            <div class="method-section">
              <h5>方法一：临时设置（当前会话）</h5>
              <p>在 Terminal 中运行以下命令：</p>
              <div class="code-block">
                <pre><code>export ANTHROPIC_BASE_URL="{{ currentBaseUrl }}"
export ANTHROPIC_AUTH_TOKEN="你的API密钥"</code></pre>
              </div>
            </div>

            <div class="method-section">
              <h5>方法二：永久设置</h5>
              <p>编辑你的 shell 配置文件（根据你使用的 shell）：</p>
              <div class="code-block">
                <pre><code># 对于 zsh (默认)
echo 'export ANTHROPIC_BASE_URL="{{ currentBaseUrl }}"' >> ~/.zshrc
echo 'export ANTHROPIC_AUTH_TOKEN="你的API密钥"' >> ~/.zshrc
source ~/.zshrc

# 对于 bash
echo 'export ANTHROPIC_BASE_URL="{{ currentBaseUrl }}"' >> ~/.bash_profile  
echo 'export ANTHROPIC_AUTH_TOKEN="你的API密钥"' >> ~/.bash_profile
source ~/.bash_profile</code></pre>
              </div>
            </div>

            <div class="method-section">
              <h5>方法三：永久设置</h5>
              <p>创建并配置 Settings 文件： 在 ~/.claude/settings.json 文件并配置您的 API 密钥</p>
              <div class="code-block">
                <pre><code>{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "your-api-key-here",
    "ANTHROPIC_BASE_URL": "{{ currentBaseUrl }}",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC":1

  },
  "permissions": {
    "allow": [],
    "deny": []
  },
  "apiKeyHelper": "echo 'your-api-key-here'"
}</code></pre>
              </div>
            </div>
          </div>
        </t-card>
      </div>
    </div>

    <!-- Linux 教程 -->
    <div v-else-if="activeTutorialSystem === 'linux'" class="tutorial-content">
      <div class="tutorial-step">
        <h3 class="step-title">
          <span class="step-number">1</span>
          安装 Node.js 环境
        </h3>

        <t-card class="step-card">
          <div class="step-content">
            <h4 class="method-title">
              <t-icon name="logo-codepen" />
              Linux 安装方法
            </h4>

            <div class="method-section">
              <h5>方法一：使用官方仓库（推荐）</h5>
              <div class="code-block">
                <pre><code># 添加 NodeSource 仓库
curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -

# 安装 Node.js
sudo apt-get install -y nodejs</code></pre>
              </div>
            </div>

            <div class="method-section">
              <h5>方法二：使用系统包管理器</h5>
              <div class="code-block">
                <pre><code># Ubuntu/Debian
sudo apt update
sudo apt install nodejs npm

# CentOS/RHEL/Fedora
sudo dnf install nodejs npm</code></pre>
              </div>
            </div>
          </div>
        </t-card>
      </div>

      <div class="tutorial-step">
        <h3 class="step-title">
          <span class="step-number">2</span>
          安装 Claude Code
        </h3>

        <t-card class="step-card">
          <div class="step-content">
            <p>打开终端，运行以下命令：</p>
            <div class="code-block">
              <pre><code># 全局安装 Claude Code
npm install -g @anthropic-ai/claude-code

# 如果遇到权限问题，使用 sudo
sudo npm install -g @anthropic-ai/claude-code</code></pre>
            </div>
          </div>
        </t-card>
      </div>

      <div class="tutorial-step">
        <h3 class="step-title">
          <span class="step-number">3</span>
          设置环境变量
        </h3>

        <t-card class="step-card">
          <div class="step-content">
            <div class="method-section">
              <h5>临时设置（当前会话）</h5>
              <div class="code-block">
                <pre><code>export ANTHROPIC_BASE_URL="{{ currentBaseUrl }}"
export ANTHROPIC_AUTH_TOKEN="你的API密钥"</code></pre>
              </div>
            </div>

            <div class="method-section">
              <h5>永久设置</h5>
              <div class="code-block">
                <pre><code># 对于 bash (默认)
echo 'export ANTHROPIC_BASE_URL="{{ currentBaseUrl }}"' >> ~/.bashrc
echo 'export ANTHROPIC_AUTH_TOKEN="你的API密钥"' >> ~/.bashrc
source ~/.bashrc

# 对于 zsh
echo 'export ANTHROPIC_BASE_URL="{{ currentBaseUrl }}"' >> ~/.zshrc
echo 'export ANTHROPIC_AUTH_TOKEN="你的API密钥"' >> ~/.zshrc
source ~/.zshrc</code></pre>
              </div>
            </div>
          </div>
        </t-card>
      </div>
    </div>

    <!-- 结尾 -->
    <div class="completion-card">
      <t-card>
        <div class="completion-content">
          <h3>🎉 恭喜你！</h3>
          <p class="completion-text">你已经成功安装并配置了 Claude Code，现在可以开始享受 AI 编程助手带来的便利了。</p>
        </div>
      </t-card>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';

// 当前系统选择
const activeTutorialSystem = ref('macos');

// 支持的系统列表
const supportedSystems = [
  { key: 'windows', name: 'Windows', icon: 'logo-windows' },
  { key: 'macos', name: 'macOS', icon: 'logo-apple' },
  { key: 'linux', name: 'Linux / WSL2', icon: 'logo-codepen' },
];

// 获取 API 基础地址
const getBaseUrlPrefix = () => {
  const apiUrl = import.meta.env.VITE_API_URL;
  if (!apiUrl || apiUrl === '/') {
    let origin = '';

    if (window.location.origin) {
      origin = window.location.origin;
    } else {
      const protocol = window.location.protocol;
      const hostname = window.location.hostname;
      const port = window.location.port;

      origin = `${protocol}//${hostname}`;

      if (port && ((protocol === 'http:' && port !== '80') || (protocol === 'https:' && port !== '443'))) {
        origin += `:${port}`;
      }
    }

    if (!origin) {
      const currentUrl = window.location.href;
      const pathStart = currentUrl.indexOf('/', 8);
      if (pathStart !== -1) {
        origin = currentUrl.substring(0, pathStart);
      } else {
        console.warn('无法获取完整的 origin，将使用相对路径');
        return '';
      }
    }

    return origin;
  }
  return apiUrl;
};

// 当前基础URL - Claude Code
const currentBaseUrl = computed(() => {
  return `${getBaseUrlPrefix()}/claude-code`;
});
</script>
<style lang="less" scoped>
.help-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;

  .help-header {
    text-align: center;
    margin-bottom: 32px;

    .help-title {
      font-size: 32px;
      font-weight: bold;
      color: var(--td-text-color-primary);
      margin-bottom: 12px;
    }

    .help-subtitle {
      font-size: 16px;
      color: var(--td-text-color-secondary);
    }
  }

  .system-tabs {
    margin-top: -20px;
    margin-bottom: 20px;

    .tabs-wrapper {
      display: flex;
      justify-content: center;
      gap: 12px;

      .system-tab-btn {
        min-width: 120px;
      }
    }
  }

  .tutorial-content {
    animation: fadeIn 0.3s ease-in-out;

    .tutorial-step {
      margin-bottom: 40px;

      .step-title {
        display: flex;
        align-items: center;
        font-size: 24px;
        font-weight: bold;
        color: var(--td-text-color-primary);
        margin-bottom: 16px;

        .step-number {
          display: inline-flex;
          align-items: center;
          justify-content: center;
          width: 32px;
          height: 32px;
          background: var(--td-brand-color);
          color: white;
          border-radius: 50%;
          font-size: 16px;
          margin-right: 12px;
        }
      }

      .step-description {
        font-size: 16px;
        color: var(--td-text-color-secondary);
        margin-bottom: 20px;
        line-height: 1.6;
      }

      .step-card,
      .verify-card {
        margin-bottom: 20px;

        .step-content {
          .method-title {
            display: flex;
            align-items: center;
            font-size: 18px;
            font-weight: bold;
            margin-bottom: 16px;
            color: var(--td-text-color-primary);

            .t-icon {
              margin-right: 8px;
            }
          }

          .method-section {
            margin-bottom: 24px;

            h5 {
              font-size: 16px;
              font-weight: bold;
              margin-bottom: 12px;
              color: var(--td-text-color-primary);
            }

            .install-steps {
              padding-left: 20px;
              line-height: 1.8;

              li {
                margin-bottom: 4px;

                code {
                  background: var(--td-bg-color-component);
                  padding: 2px 6px;
                  border-radius: 4px;
                  font-family: 'Monaco', 'Menlo', monospace;
                  font-size: 14px;
                }
              }
            }
          }

          .note-text {
            font-size: 14px;
            color: var(--td-warning-color);
            margin-top: 8px;
          }
        }

        .verify-title {
          font-size: 18px;
          font-weight: bold;
          margin-bottom: 12px;
          color: var(--td-success-color);
        }

        .verify-note {
          font-size: 14px;
          color: var(--td-success-color);
          margin-top: 8px;
        }
      }
    }
  }

  .code-block {
    background: var(--td-bg-color-container);
    border: 1px solid var(--td-component-stroke);
    border-radius: 6px;
    margin: 12px 0;

    pre {
      margin: 0;
      padding: 16px;
      font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
      font-size: 14px;
      line-height: 1.5;
      color: var(--td-text-color-primary);
      overflow-x: auto;
    }
  }

  .env-vars {
    margin-top: 12px;

    .env-var {
      background: var(--td-bg-color-container);
      padding: 12px;
      border-radius: 4px;
      margin-bottom: 8px;
      font-size: 14px;

      code {
        background: var(--td-bg-color-component);
        padding: 2px 6px;
        border-radius: 4px;
        font-family: 'Monaco', 'Menlo', monospace;
      }
    }
  }

  .expected-output {
    margin-top: 20px;

    .output-example {
      background: var(--td-bg-color-container);
      padding: 12px;
      border-radius: 4px;
      font-family: 'Monaco', 'Menlo', monospace;
      font-size: 14px;
      margin: 8px 0;
    }
  }

  .verify-section {
    margin-bottom: 20px;

    h5 {
      font-size: 16px;
      font-weight: bold;
      margin-bottom: 8px;
    }
  }

  .usage-section {
    margin-bottom: 20px;

    h5 {
      font-size: 16px;
      font-weight: bold;
      margin-bottom: 8px;
    }
  }

  .troubleshooting {
    :deep(.t-collapse) {
      .t-collapse-panel {
        margin-bottom: 8px;
      }

      .t-collapse-panel__content {
        padding: 16px;

        ul {
          padding-left: 20px;

          li {
            margin-bottom: 4px;

            code {
              background: var(--td-bg-color-component);
              padding: 2px 6px;
              border-radius: 4px;
              font-family: 'Monaco', 'Menlo', monospace;
              font-size: 14px;
            }
          }
        }
      }
    }
  }

  .completion-card {
    margin-top: 40px;

    .completion-content {
      text-align: center;
      padding: 20px;

      h3 {
        font-size: 24px;
        margin-bottom: 16px;
        color: var(--td-text-color-primary);
      }

      .completion-text {
        font-size: 16px;
        color: var(--td-text-color-secondary);
        margin-bottom: 12px;
        line-height: 1.6;
      }

      .completion-note {
        font-size: 14px;
        color: var(--td-text-color-placeholder);
        line-height: 1.5;
      }
    }
  }
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

code {
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
}

.note-alert {
  margin-top: 16px;

  h6 {
    font-weight: bold;
    margin-bottom: 8px;
  }

  ul {
    margin: 0;
    padding-left: 20px;

    li {
      margin-bottom: 4px;
    }
  }
}
</style>
