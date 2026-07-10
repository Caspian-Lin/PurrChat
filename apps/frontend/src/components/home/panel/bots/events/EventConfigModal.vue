<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="$emit('close')">
      <div class="modal-container">
        <!-- 头部 -->
        <div class="modal-header">
          <h3 class="modal-title">{{ isEditing ? '编辑事件' : '新建事件' }}</h3>
          <button class="modal-close" @click="$emit('close')">
            <BsX :size="18" />
          </button>
        </div>

        <!-- 类型选择（仅新建时） -->
        <div v-if="!isEditing" class="modal-body">
          <template v-for="group in nodeTypeGroups" :key="group.category">
            <div class="type-section-label">{{ group.label }}</div>
            <div class="type-selector">
              <button
                v-for="node in group.nodes"
                :key="node.type"
                class="type-card"
                :class="{ 'type-card--active': form.type === node.type }"
                @click="selectType(node.type)"
              >
                <span class="type-card__icon">{{ node.icon }}</span>
                <span class="type-card__label">{{ node.label }}</span>
              </button>
            </div>
          </template>
        </div>

        <!-- 配置表单 -->
        <div class="modal-body">
          <!-- 通用字段：事件名称 -->
          <div class="form-group">
            <label class="form-label">事件名称</label>
            <input v-model="form.name" type="text" class="form-input" placeholder="事件名称" />
            <p v-if="nameValidationError" class="validation-error">{{ nameValidationError }}</p>
          </div>

          <!-- trigger：仅名称，无额外配置 -->

          <!-- end：仅名称 -->

          <!-- wait 配置 -->
          <template v-if="form.type === 'wait'">
            <div class="form-group">
              <label class="form-label">等待类型</label>
              <select v-model="form.config.wait_type" class="form-input">
                <option value="user_message">用户消息</option>
                <option value="custom">自定义条件</option>
              </select>
            </div>
            <div v-if="form.config.wait_type === 'custom'" class="form-group">
              <label class="form-label">条件表达式</label>
              <textarea
                v-model="form.config.condition"
                class="form-textarea"
                rows="3"
                placeholder="输入等待条件表达式..."
              />
            </div>
          </template>

          <!-- if 配置 -->
          <template v-if="form.type === 'if'">
            <div class="form-group">
              <label class="form-label">条件逻辑</label>
              <div class="logic-toggle">
                <button
                  class="logic-toggle__btn"
                  :class="{ 'logic-toggle__btn--active': ifLogic === 'and' }"
                  type="button"
                  @click="ifLogic = 'and'"
                >
                  AND（全部满足）
                </button>
                <button
                  class="logic-toggle__btn"
                  :class="{ 'logic-toggle__btn--active': ifLogic === 'or' }"
                  type="button"
                  @click="ifLogic = 'or'"
                >
                  OR（任一满足）
                </button>
              </div>
            </div>
            <div class="form-group">
              <label class="form-label">条件列表</label>
              <div class="if-conditions">
                <div v-for="(cond, idx) in ifConditions" :key="idx" class="if-condition-row">
                  <span class="if-condition-row__index">{{ idx + 1 }}</span>
                  <div class="if-condition-row__input-wrap">
                    <input
                      :ref="
                        (el: any) => {
                          if (el) ifCondRefs[`${idx}-left`] = el;
                        }
                      "
                      v-model="cond.left"
                      type="text"
                      class="form-input if-condition-row__input"
                      placeholder="{节点.端口}"
                    />
                    <button
                      class="var-insert-btn var-insert-btn--small"
                      type="button"
                      title="插入变量"
                      @click="(e: MouseEvent) => openVarPicker(e, ifCondRefs[`${idx}-left`])"
                    >
                      {'{ }'}
                    </button>
                  </div>
                  <select v-model="cond.operator" class="form-input if-condition-row__operator">
                    <option value="==">==</option>
                    <option value="!=">!=</option>
                    <option value="contains">包含</option>
                    <option value=">">&gt;</option>
                    <option value="<">&lt;</option>
                    <option value="startsWith">开头是</option>
                    <option value="endsWith">结尾是</option>
                  </select>
                  <div class="if-condition-row__input-wrap">
                    <input
                      :ref="
                        (el: any) => {
                          if (el) ifCondRefs[`${idx}-right`] = el;
                        }
                      "
                      v-model="cond.right"
                      type="text"
                      class="form-input if-condition-row__input"
                      placeholder="{节点.端口} / 值"
                    />
                    <button
                      class="var-insert-btn var-insert-btn--small"
                      type="button"
                      title="插入变量"
                      @click="(e: MouseEvent) => openVarPicker(e, ifCondRefs[`${idx}-right`])"
                    >
                      {'{ }'}
                    </button>
                  </div>
                  <button
                    v-if="ifConditions.length > 1"
                    class="port-remove-btn"
                    type="button"
                    @click="ifConditions.splice(idx, 1)"
                  >
                    &times;
                  </button>
                </div>
              </div>
              <button
                class="add-port-btn"
                type="button"
                @click="ifConditions.push({ left: '', operator: '==', right: '' })"
              >
                + 添加条件
              </button>
              <p class="form-hint">
                条件值支持变量引用格式 <code>{'{节点名.端口名}'}</code>，点击输入框右侧的
                <code>{'{ }'}</code> 按钮选择。
              </p>
            </div>
          </template>

          <!-- loop 配置 -->
          <template v-if="form.type === 'loop'">
            <p class="form-hint" style="margin-bottom: 12px">
              循环节点有两个出口：「循环体」连接循环体起始节点，「完成」连接循环结束后的节点。
              循环体末尾需要连回循环节点的「执行」入口形成回环。
            </p>
            <div class="form-group">
              <label class="form-label">条件表达式</label>
              <textarea
                v-model="form.config.condition"
                class="form-textarea"
                rows="2"
                placeholder="true / false / $变量引用"
              />
              <p class="form-hint">每轮迭代前求值，为 false 时退出循环。留空则视为 false。</p>
            </div>
            <div class="form-group">
              <label class="form-label">最大迭代次数</label>
              <input
                v-model.number="form.config.max_iterations"
                type="number"
                min="1"
                class="form-input"
              />
              <p class="form-hint">
                安全上限，防止无限循环。循环内可用变量：
                <code>$loopID:loop_index</code>（当前序号）、
                <code>$loopID:loop_iterations</code>（总次数）
              </p>
            </div>
          </template>

          <!-- switch 配置 -->
          <template v-if="form.type === 'switch'">
            <p class="form-hint">根据匹配值路由到不同分支。将上游数据连接到「匹配值」输入端口。</p>
            <div class="form-group">
              <label class="form-label">分支列表</label>
              <div class="switch-cases">
                <div v-for="(c, idx) in switchCases" :key="idx" class="switch-case-row">
                  <input
                    v-model="c.label"
                    placeholder="分支名称"
                    class="form-input"
                    style="flex: 1"
                  />
                  <input
                    v-model="c.value"
                    placeholder="匹配值（留空则跳过）"
                    class="form-input"
                    style="flex: 1"
                  />
                  <button
                    v-if="switchCases.length > 2"
                    class="port-remove-btn"
                    @click="switchCases.splice(idx, 1)"
                  >
                    ×
                  </button>
                </div>
              </div>
              <button
                class="add-port-btn"
                @click="switchCases.push({ value: '', label: `分支 ${switchCases.length + 1}` })"
              >
                + 添加分支
              </button>
            </div>
          </template>

          <!-- merge 配置 -->
          <template v-if="form.type === 'merge'">
            <p class="form-hint">汇聚多条分支到同一个后续节点。调整输入数量后需重新连接。</p>
            <div class="form-group">
              <label class="form-label">输入数量</label>
              <input
                v-model.number="form.config.input_count"
                type="number"
                min="2"
                max="10"
                class="form-input"
              />
            </div>
          </template>

          <!-- tool 配置 -->
          <template v-if="form.type === 'tool'">
            <div class="form-group">
              <label class="form-label">请求方法</label>
              <select v-model="form.config.method" class="form-input">
                <option value="GET">GET</option>
                <option value="POST">POST</option>
                <option value="PUT">PUT</option>
                <option value="DELETE">DELETE</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">URL</label>
              <input
                v-model="form.config.url"
                type="text"
                class="form-input"
                placeholder="https://api.example.com/data"
              />
            </div>
            <div class="form-group">
              <label class="form-label">Headers（JSON）</label>
              <textarea
                v-model="form.config.headers"
                class="form-textarea form-textarea--code"
                rows="2"
                placeholder='{"Authorization": "Bearer ..."}'
              />
            </div>
            <div v-if="form.config.method !== 'GET'" class="form-group">
              <label class="form-label">请求体</label>
              <div class="form-input-with-action">
                <textarea
                  ref="toolBodyTextarea"
                  v-model="form.config.body"
                  class="form-textarea form-textarea--code"
                  rows="3"
                  placeholder="请求体内容（可使用变量）"
                />
                <button
                  class="var-insert-btn"
                  title="插入变量"
                  @click="(e) => openVarPicker(e, $refs.toolBodyTextarea as HTMLTextAreaElement)"
                >
                  {'{ }'}
                </button>
              </div>
            </div>
            <div class="form-group">
              <label class="form-label">超时 (ms)</label>
              <input
                v-model.number="form.config.timeout_ms"
                type="number"
                min="1000"
                max="30000"
                step="1000"
                class="form-input"
              />
            </div>
            <p class="form-hint">响应内容输出到「响应」端口，HTTP 状态码输出到「状态码」端口。</p>
          </template>

          <!-- dify 配置 -->
          <template v-if="form.type === 'dify'">
            <div class="form-group">
              <label class="form-label">应用类型</label>
              <select v-model="form.config.app_type" class="form-input">
                <option value="workflow">Workflow（单次执行）</option>
                <option value="chatflow">Chatflow（多轮对话）</option>
              </select>
              <p class="form-hint">
                Workflow 适合翻译、摘要等单次处理；Chatflow 支持多轮上下文记忆。
              </p>
            </div>
            <div class="form-group">
              <label class="form-label">API 地址</label>
              <input
                v-model="form.config.api_base"
                type="text"
                class="form-input"
                placeholder="https://api.dify.ai/v1"
              />
            </div>
            <div class="form-group">
              <label class="form-label">API Key</label>
              <input
                v-model="form.config.api_key"
                type="password"
                class="form-input"
                placeholder="app-..."
              />
            </div>
            <div class="form-group">
              <label class="form-label">输入变量映射（JSON）</label>
              <div class="form-input-with-action">
                <textarea
                  ref="difyInputsTextarea"
                  v-model="form.config.inputs_mapping"
                  class="form-textarea form-textarea--code"
                  rows="2"
                  placeholder='{"query": "$triggerID:out_input"}'
                />
                <button
                  class="var-insert-btn"
                  title="插入变量"
                  @click="(e) => openVarPicker(e, $refs.difyInputsTextarea as HTMLTextAreaElement)"
                >
                  {'{ }'}
                </button>
              </div>
              <p class="form-hint">
                键为 Dify 工作流变量名，值支持变量引用。点击
                <strong>{'{}'}</strong> 从上游选择变量。
              </p>
            </div>
            <div class="form-group">
              <label class="form-label">输出路径</label>
              <input
                v-model="form.config.output_path"
                type="text"
                class="form-input"
                placeholder="data.outputs.text"
              />
              <p class="form-hint">
                从响应 JSON 中提取字段的路径。留空则自动提取 data.outputs（单字段时直接取值）。
              </p>
            </div>
            <div class="form-group">
              <label class="form-label">响应模式</label>
              <select v-model="form.config.response_mode" class="form-input">
                <option value="blocking">Blocking（等待结果）</option>
                <option value="streaming">Streaming（仅取最终结果）</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">超时 (ms)</label>
              <input
                v-model.number="form.config.timeout_ms"
                type="number"
                min="1000"
                max="120000"
                step="1000"
                class="form-input"
              />
            </div>
            <p class="form-hint">输出到「输出」端口，错误信息输出到「错误」端口。</p>
          </template>

          <!-- n8n 配置 -->
          <template v-if="form.type === 'n8n'">
            <div class="form-group">
              <label class="form-label">Webhook URL（生产环境）</label>
              <input
                v-model="form.config.webhook_url"
                type="text"
                class="form-input"
                placeholder="https://your-n8n.com/webhook/abc123"
              />
            </div>
            <div class="form-group">
              <label class="form-label">HTTP 方法</label>
              <select v-model="form.config.method" class="form-input">
                <option value="POST">POST</option>
                <option value="GET">GET</option>
              </select>
            </div>
            <div v-if="form.config.method === 'POST'" class="form-group">
              <label class="form-label">请求体</label>
              <div class="form-input-with-action">
                <textarea
                  ref="n8nBodyTextarea"
                  v-model="form.config.body"
                  class="form-textarea form-textarea--code"
                  rows="3"
                  placeholder='{"input": "$triggerID:out_input"}'
                />
                <button
                  class="var-insert-btn"
                  title="插入变量"
                  @click="(e) => openVarPicker(e, $refs.n8nBodyTextarea as HTMLTextAreaElement)"
                >
                  {'{ }'}
                </button>
              </div>
              <p class="form-hint">留空时自动使用「输入」端口的值作为请求体。</p>
            </div>
            <div class="form-group">
              <label class="form-label">认证方式</label>
              <select v-model="form.config.auth_type" class="form-input">
                <option value="none">无</option>
                <option value="header">Header Auth</option>
                <option value="basic">Basic Auth</option>
              </select>
            </div>
            <div v-if="form.config.auth_type !== 'none'" class="form-group">
              <label class="form-label">
                {{ form.config.auth_type === 'basic' ? '用户名:密码' : 'Header 名称:值' }}
              </label>
              <input
                v-model="form.config.auth_credential"
                type="text"
                class="form-input"
                :placeholder="
                  form.config.auth_type === 'basic' ? 'user:password' : 'X-API-Key:your-key'
                "
              />
              <p class="form-hint">
                {{
                  form.config.auth_type === 'basic' ? '格式: 用户名:密码' : '格式: Header名称:值'
                }}
              </p>
            </div>
            <div class="form-group">
              <label class="form-label">超时 (ms)</label>
              <input
                v-model.number="form.config.timeout_ms"
                type="number"
                min="1000"
                max="60000"
                step="1000"
                class="form-input"
              />
            </div>
            <p class="form-hint">输出到「输出」端口，错误信息输出到「错误」端口。</p>
          </template>

          <!-- LLM 配置 -->
          <template v-if="form.type === 'llm'">
            <!-- 从 AI 面板导入配置 -->
            <div v-if="aiStore.configs.length" class="form-group">
              <label class="form-label">复用 AI 面板配置</label>
              <select
                class="form-input"
                style="cursor: pointer; appearance: none"
                @change="importFromAiPanel(($event.target as HTMLSelectElement).value)"
              >
                <option value="" disabled selected>选择配置...</option>
                <option v-for="cfg in aiStore.configs" :key="cfg.id" :value="cfg.id">
                  {{ cfg.name }}
                </option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">API URL</label>
              <input
                v-model="form.config.api_url"
                type="text"
                class="form-input"
                placeholder="https://api.openai.com/v1/chat/completions"
              />
            </div>
            <div class="form-group">
              <label class="form-label">API Key</label>
              <input
                v-model="form.config.api_key"
                type="password"
                class="form-input"
                placeholder="sk-..."
              />
            </div>
            <div class="form-row">
              <div class="form-group">
                <label class="form-label">模型</label>
                <input
                  v-model="form.config.model"
                  type="text"
                  class="form-input"
                  placeholder="gpt-4o"
                />
              </div>
              <div class="form-group">
                <label class="form-label">Temperature</label>
                <input
                  v-model.number="form.config.temperature"
                  type="number"
                  step="0.1"
                  min="0"
                  max="2"
                  class="form-input"
                />
              </div>
            </div>
            <div class="form-group">
              <label class="form-label">System Prompt</label>
              <textarea
                v-model="form.config.system_prompt"
                class="form-textarea"
                rows="3"
                placeholder="你是一个..."
              />
            </div>
            <div class="form-row">
              <div class="form-group">
                <label class="form-label">Max Tokens</label>
                <input
                  v-model.number="form.config.max_tokens"
                  type="number"
                  min="1"
                  class="form-input"
                />
              </div>
              <div class="form-group">
                <label class="form-label">上下文窗口</label>
                <input
                  v-model.number="form.config.context_window"
                  type="number"
                  min="1"
                  class="form-input"
                />
              </div>
            </div>
          </template>

          <!-- 内置事件配置 -->
          <template v-if="form.type === 'builtin'">
            <div class="form-group">
              <label class="form-label">内置类型</label>
              <select v-model="form.config.builtin_type" class="form-input">
                <option v-for="bt in builtinTypes" :key="bt.value" :value="bt.value">
                  {{ bt.label }}
                </option>
              </select>
            </div>
            <div v-if="form.config.builtin_type === 'random_number'" class="form-row">
              <div class="form-group">
                <label class="form-label">最小值</label>
                <input v-model.number="form.config.min" type="number" class="form-input" />
              </div>
              <div class="form-group">
                <label class="form-label">最大值</label>
                <input v-model.number="form.config.max" type="number" class="form-input" />
              </div>
            </div>
            <div v-if="form.config.builtin_type === 'echo'" class="form-row">
              <div class="form-group">
                <label class="form-label">前缀</label>
                <input v-model="form.config.prefix" type="text" class="form-input" />
              </div>
              <div class="form-group">
                <label class="form-label">后缀</label>
                <input v-model="form.config.suffix" type="text" class="form-input" />
              </div>
            </div>
            <div v-if="form.config.builtin_type === 'template'">
              <div class="form-group">
                <label class="form-label">模板</label>
                <div class="form-input-with-action">
                  <textarea
                    ref="builtinTemplateTextarea"
                    v-model="form.config.template"
                    class="form-textarea"
                    rows="3"
                    placeholder="你好，{input}！"
                  />
                  <button
                    class="var-insert-btn"
                    title="插入变量"
                    @click="
                      (e) => openVarPicker(e, $refs.builtinTemplateTextarea as HTMLTextAreaElement)
                    "
                  >
                    {'{ }'}
                  </button>
                </div>
                <p class="form-hint">
                  可用变量：{'{input}'} 当前消息 · {'{username}'} 发送者 · {'{time}'} 时间 ·
                  {'{args}'} 除首个词外的参数 · {'{args:N}'} 第 N 个词（0 起）
                </p>
              </div>
            </div>
          </template>

          <!-- Python 事件配置 -->
          <template v-if="form.type === 'python'">
            <div class="form-group">
              <label class="form-label">Python 代码</label>
              <textarea
                v-model="form.config.code"
                class="form-textarea form-textarea--code"
                rows="8"
                placeholder="def run(context, input_data):&#10;    # 处理输入&#10;    result = input_data.get('input', '')&#10;    return {'result': result}"
                spellcheck="false"
              />
            </div>
            <div class="form-group">
              <label class="form-label">超时 (ms)</label>
              <input
                v-model.number="form.config.timeout_ms"
                type="number"
                min="1000"
                max="30000"
                step="1000"
                class="form-input"
              />
            </div>
          </template>

          <!-- template 配置 -->
          <template v-if="form.type === 'template'">
            <div class="form-group">
              <label class="form-label">模板内容</label>
              <div class="form-input-with-action">
                <textarea
                  ref="templateTextarea"
                  v-model="form.config.template"
                  class="form-textarea"
                  rows="4"
                  placeholder="使用 {variable} 引用变量..."
                />
                <button
                  class="var-insert-btn"
                  title="插入变量"
                  @click="(e) => openVarPicker(e, $refs.templateTextarea as HTMLTextAreaElement)"
                >
                  {'{ }'}
                </button>
              </div>
              <p class="form-hint">
                可用变量：{'{input}'} 当前消息 · {'{username}'} 发送者 · {'{time}'} 时间 ·
                {'{args}'} 除首个词外的参数 · {'{args:N}'} 第 N 个词（0 起）。 点击
                <strong>{'{}'}</strong> 按钮从上游节点选择变量。
              </p>
            </div>
          </template>

          <!-- reply 配置 -->
          <template v-if="form.type === 'reply'">
            <div class="form-group">
              <label class="form-label">回复模板</label>
              <div class="form-input-with-action">
                <textarea
                  ref="replyTextarea"
                  v-model="form.config.template"
                  class="form-textarea"
                  rows="3"
                  placeholder="使用 {节点名.端口名} 引用其他节点的输出"
                />
                <button
                  class="var-insert-btn"
                  title="插入变量"
                  @click="(e) => openVarPicker(e, $refs.replyTextarea as HTMLTextAreaElement)"
                >
                  {'{ }'}
                </button>
              </div>
              <p class="form-hint">
                可用变量：{'{input}'} 当前消息 · {'{username}'} 发送者 · {'{time}'} 时间 ·
                {'{args}'} 除首个词外的参数 · {'{args:N}'} 第 N 个词（0 起）。 点击
                <strong>{'{}'}</strong> 按钮从上游节点选择变量。
              </p>
            </div>
          </template>

          <!-- history 配置 -->
          <template v-if="form.type === 'history'">
            <div class="form-group">
              <label class="form-label">获取消息条数</label>
              <input
                v-model.number="form.config.count"
                type="number"
                min="1"
                max="100"
                class="form-input"
                placeholder="20"
              />
              <p class="form-hint">
                获取最近 N 条会话消息，格式化为 prompt 字符串。图片消息将显示为 [图片]。
              </p>
            </div>
          </template>

          <!-- 自定义端口 -->
          <div v-if="supportsCustomPorts" class="form-group">
            <label class="form-label">自定义端口</label>
            <div class="custom-ports">
              <div v-for="(port, idx) in customPorts" :key="idx" class="custom-port-row">
                <select v-model="port.direction" class="form-input" style="width: 70px">
                  <option value="input">输入</option>
                  <option value="output">输出</option>
                </select>
                <input v-model="port.name" placeholder="端口名称" class="form-input" />
                <select v-model="port.dataType" class="form-input" style="width: 80px">
                  <option value="string">字符串</option>
                  <option value="number">数值</option>
                  <option value="boolean">布尔</option>
                  <option value="any">任意</option>
                </select>
                <button class="port-remove-btn" @click="removeCustomPort(idx)">×</button>
              </div>
              <button class="add-port-btn" @click="addCustomPort">+ 添加端口</button>
            </div>
          </div>
        </div>

        <!-- 底部 -->
        <div class="modal-footer">
          <button v-if="isEditing" class="btn btn--danger" @click="$emit('delete', form.id)">
            删除
          </button>
          <div class="modal-footer__right">
            <button class="btn btn--ghost" @click="$emit('close')">取消</button>
            <button class="btn btn--primary" @click="handleConfirm">确认</button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>

  <!-- 变量选择器 -->
  <VarReferencePicker
    :visible="showVarPicker"
    :current-node-id="form.id"
    :events="props.existingEvents"
    :connections="props.connections"
    :anchor="varPickerAnchor"
    @close="showVarPicker = false"
    @select="handleVarSelect"
  />
</template>

<script setup lang="ts">
import { reactive, computed, watch, ref, nextTick } from 'vue';
import { BsX } from 'vue-icons-plus/bs';
import { useAiStore } from '../../../../../stores/ai';
import { useAuthStore } from '../../../../../stores/auth';
import { getDefaultPorts, type EventType } from '../../../../../utils/portTypes';
import {
  generateNodeKey,
  type EventPort,
  type WorkflowEvent,
  type FlowConnection,
} from '@purrchat/workflow-types';
import { PRODUCTION_NODE_MANIFEST } from '../../../../../utils/workflowDocument';
import VarReferencePicker from './VarReferencePicker.vue';

interface Props {
  visible: boolean;
  editingEvent?: WorkflowEvent | null;
  existingEvents?: WorkflowEvent[];
  connections?: FlowConnection[];
}

const props = withDefaults(defineProps<Props>(), {
  editingEvent: null,
  existingEvents: () => [],
  connections: () => [],
});

const emit = defineEmits<{
  close: [];
  confirm: [event: WorkflowEvent];
  delete: [eventId: string];
}>();

const isEditing = computed(() => !!props.editingEvent);

const aiStore = useAiStore();
const authStore = useAuthStore();

// 初始化 AI store（特殊模式编辑器在新标签页打开，AiPanel 不会挂载）
aiStore.initStore(authStore.currentUser?.id);

// 名称验证错误
const nameValidationError = ref('');

// If 条件配置 — 使用 computed 双向绑定到 form.config.conditions
const ifCondRefs = reactive<Record<string, HTMLInputElement>>({});

const ifConditions = computed({
  get: () => {
    const raw = form.config.conditions;
    if (Array.isArray(raw) && raw.length > 0)
      return raw as { left: string; operator: string; right: string }[];
    // 旧格式迁移：从 operator/left_default/right_default 转换
    return [
      {
        left: (form.config.left_default as string) || '',
        operator: (form.config.operator as string) || '==',
        right: (form.config.right_default as string) || '',
      },
    ];
  },
  set: (val) => {
    form.config.conditions = val;
    // 清理旧字段
    delete form.config.operator;
    delete form.config.left_default;
    delete form.config.right_default;
  },
});

const ifLogic = computed({
  get: () => ((form.config.logic as string) || 'and').toLowerCase(),
  set: (val) => {
    form.config.logic = val;
  },
});

// 变量选择器状态
const showVarPicker = ref(false);
const varPickerAnchor = ref({ x: 0, y: 0 });
const varPickerTarget = ref<HTMLTextAreaElement | HTMLInputElement | null>(null);

function openVarPicker(event: MouseEvent, targetEl: HTMLTextAreaElement | HTMLInputElement) {
  varPickerTarget.value = targetEl;
  varPickerAnchor.value = {
    x: (event.target as HTMLElement).getBoundingClientRect().left,
    y: (event.target as HTMLElement).getBoundingClientRect().bottom,
  };
  showVarPicker.value = true;
}

function handleVarSelect(item: { value: string; ref: string }) {
  if (!varPickerTarget.value) return;
  const el = varPickerTarget.value;
  const start = el.selectionStart ?? el.value.length;
  const end = el.selectionEnd ?? start;
  const before = el.value.substring(0, start);
  const after = el.value.substring(end);
  const insertText = item.ref; // 使用人类可读格式 {nodeName.portName}
  el.value = before + insertText + after;
  // 触发 v-model 更新
  el.dispatchEvent(new Event('input', { bubbles: true }));
  // 将光标移动到插入内容之后
  nextTick(() => {
    const newPos = start + insertText.length;
    el.focus();
    el.setSelectionRange(newPos, newPos);
  });
}

// 自定义端口列表
const customPorts = reactive<{ name: string; dataType: string; direction: 'input' | 'output' }[]>(
  []
);

// Switch 分支列表
const switchCases = computed({
  get: () => (form.config.cases as { value: string; label: string }[]) || [],
  set: (val) => {
    form.config.cases = val;
  },
});

const categoryLabels = {
  control: '控制流',
  process: '处理',
  output: '输出',
} as const;

const nodeTypeGroups = (['control', 'process', 'output'] as const)
  .map((category) => ({
    category,
    label: categoryLabels[category],
    nodes: PRODUCTION_NODE_MANIFEST.filter((node) => node.category === category),
  }))
  .filter((group) => group.nodes.length > 0);

// 是否为支持自定义端口的节点类型
const supportsCustomPorts = computed(() =>
  [
    'llm',
    'builtin',
    'python',
    'template',
    'if',
    'wait',
    'reply',
    'history',
    'dify',
    'n8n',
  ].includes(form.type)
);

const builtinTypes = [
  { value: 'random_number', label: '随机数' },
  { value: 'haiku', label: '俳句' },
  { value: 'echo', label: '回显' },
  { value: 'count', label: '计数器' },
  { value: 'template', label: '模板' },
];

function getDefaultConfig(type: EventType): Record<string, any> {
  const manifest = PRODUCTION_NODE_MANIFEST.find((node) => node.type === type);
  return manifest ? structuredClone(manifest.defaultConfig) : {};
}

interface FormData {
  id: string;
  key?: string;
  type: EventType;
  name: string;
  config: Record<string, any>;
  ports: EventPort[];
  position?: { x: number; y: number };
}

const form = reactive<FormData>({
  id: '',
  type: PRODUCTION_NODE_MANIFEST[0].type,
  name: '',
  config: getDefaultConfig(PRODUCTION_NODE_MANIFEST[0].type),
  ports: getDefaultPorts(PRODUCTION_NODE_MANIFEST[0].type),
});

// 选择类型时重置 config、ports 和名称
function selectType(type: EventType) {
  form.type = type;
  form.config = getDefaultConfig(type);
  form.ports = getDefaultPorts(type);
  form.name = PRODUCTION_NODE_MANIFEST.find((node) => node.type === type)?.label || type;
  customPorts.length = 0;
  nameValidationError.value = '';
}

function importFromAiPanel(configId: string) {
  const config = aiStore.configs.find((c) => c.id === configId);
  if (!config) return;
  form.config.api_url = config.apiUrl;
  form.config.api_key = config.apiKey;
  form.config.model = config.model;
  form.config.temperature = config.temperature;
  if (config.maxTokens) form.config.max_tokens = config.maxTokens;
}

// 自定义端口操作
function addCustomPort() {
  customPorts.push({ name: '', dataType: 'string', direction: 'input' });
}

function removeCustomPort(idx: number) {
  customPorts.splice(idx, 1);
}

// 从已有事件的 ports 中提取自定义端口
function extractCustomPorts(
  event: WorkflowEvent
): { name: string; dataType: string; direction: 'input' | 'output' }[] {
  if (!event.ports) return [];
  const defaultPorts = getDefaultPorts(event.type);
  const defaultIds = new Set(defaultPorts.map((p) => p.id));
  return event.ports
    .filter((p) => !defaultIds.has(p.id))
    .map((p) => ({ name: p.name, dataType: p.dataType, direction: p.direction }));
}

watch(
  () => props.visible,
  () => {
    if (props.visible && props.editingEvent) {
      Object.assign(form, {
        id: props.editingEvent.id,
        key: props.editingEvent.key,
        type: props.editingEvent.type,
        name: props.editingEvent.name,
        config: structuredClone(props.editingEvent.config),
        ports: [...(props.editingEvent.ports || getDefaultPorts(props.editingEvent.type))],
        position: props.editingEvent.position ? { ...props.editingEvent.position } : undefined,
      });
      // 恢复自定义端口
      customPorts.length = 0;
      const extracted = extractCustomPorts(props.editingEvent);
      extracted.forEach((p) => customPorts.push(p));
    } else if (props.visible) {
      const type = PRODUCTION_NODE_MANIFEST[0].type;
      Object.assign(form, {
        id: `evt_${Date.now()}`,
        key: undefined,
        type,
        name: '',
        config: getDefaultConfig(type),
        ports: getDefaultPorts(type),
        position: undefined,
      });
      customPorts.length = 0;
    }
    nameValidationError.value = '';
  }
);

function handleConfirm() {
  // 验证名称
  if (!form.name.trim()) {
    nameValidationError.value = '事件名称不能为空';
    return;
  }
  nameValidationError.value = '';

  // 合并默认端口和自定义端口
  const defaultPorts = getDefaultPorts(form.type);
  const customEventPorts: EventPort[] = customPorts
    .filter((p) => p.name.trim())
    .map((p) => ({
      id: `${p.direction === 'output' ? 'out_custom' : 'in_custom'}_${p.name}`,
      name: p.name,
      dataType: p.dataType as EventPort['dataType'],
      direction: p.direction,
    }));

  const event: WorkflowEvent = {
    id: form.id,
    type: form.type,
    name: form.name,
    key:
      form.key ||
      (() => {
        const existingKeys = new Set(props.existingEvents.map((item) => item.key).filter(Boolean));
        let index = 1;
        let key = generateNodeKey(form.type, index);
        while (existingKeys.has(key)) key = generateNodeKey(form.type, ++index);
        return key;
      })(),
    config: { ...form.config },
    ports: [...defaultPorts, ...customEventPorts],
  };

  if (form.position) {
    event.position = { ...form.position };
  }

  emit('confirm', event);
  emit('close');
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: var(--modal-overlay-color, rgba(0, 0, 0, 0.4));
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
  backdrop-filter: blur(2px);
}

.modal-container {
  background: var(--strong-background-color, #fff);
  border-radius: var(--radius-lg, 12px);
  width: 520px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-lg, 0 8px 32px rgba(28, 25, 23, 0.1));
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.06));
}

.modal-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-color, #1c1917);
}

.modal-close {
  padding: 4px;
  border-radius: var(--radius-xs, 4px);
  color: var(--text-tertiary-color, #a8a29e);
  cursor: pointer;
  transition: color 0.15s;
}
.modal-close:hover {
  color: var(--text-color, #1c1917);
  background: var(--surface-tertiary-color, #e8e4de);
}

.modal-body {
  padding: 16px 20px;
  overflow-y: auto;
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  border-top: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.06));
}

.modal-footer__right {
  display: flex;
  gap: 8px;
}

.type-section-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary-color, #a8a29e);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 8px;
}

.type-section-label:not(:first-child) {
  margin-top: 16px;
}

.type-selector {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
}

.type-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 12px 8px;
  border-radius: var(--radius-sm, 8px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.06));
  cursor: pointer;
  transition: all 0.15s;
  background: none;
  color: var(--text-secondary-color, #57534e);
}
.type-card:hover {
  background: var(--surface-tertiary-color, #e8e4de);
}
.type-card--active {
  border-color: var(--theme-primary, #5a8f4e);
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 6%, transparent);
}

.type-card__icon {
  font-size: 24px;
}

.type-card__label {
  font-size: 12px;
}

.form-group {
  margin-bottom: 12px;
}

.form-label {
  display: block;
  font-size: 12px;
  color: var(--text-secondary-color, #57534e);
  margin-bottom: 4px;
}

.form-input {
  width: 100%;
  padding: 8px 10px;
  font-size: 13px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.1));
  background: var(--input-background, #fff);
  color: var(--text-color, #1c1917);
  outline: none;
  transition: border-color 0.15s;
  box-sizing: border-box;
}
.form-input:focus {
  border-color: var(--theme-primary, #5a8f4e);
}

.form-textarea {
  width: 100%;
  padding: 8px 10px;
  font-size: 13px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.1));
  background: var(--input-background, #fff);
  color: var(--text-color, #1c1917);
  outline: none;
  resize: vertical;
  font-family: inherit;
  transition: border-color 0.15s;
  box-sizing: border-box;
}
.form-textarea:focus {
  border-color: var(--theme-primary, #5a8f4e);
}

.form-textarea--code {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 12px;
  line-height: 1.5;
  tab-size: 2;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.form-hint {
  font-size: 11px;
  color: var(--text-tertiary-color, #a8a29e);
  margin-top: 4px;
}

/* 自定义端口 */
.custom-ports {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.custom-port-row {
  display: grid;
  grid-template-columns: 70px 1fr 80px 32px;
  gap: 6px;
  align-items: center;
}

.add-port-btn {
  padding: 4px 10px;
  font-size: 11px;
  border-radius: var(--radius-xs, 4px);
  border: 1px dashed var(--border-subtle-color, rgba(0, 0, 0, 0.1));
  color: var(--text-tertiary-color, #a8a29e);
  cursor: pointer;
  background: none;
  transition: all 0.15s;
}
.add-port-btn:hover {
  border-color: var(--theme-primary, #5a8f4e);
  color: var(--theme-primary, #5a8f4e);
}

.port-remove-btn {
  width: 24px;
  height: 24px;
  border-radius: var(--radius-xs, 4px);
  border: none;
  background: none;
  color: var(--text-tertiary-color, #a8a29e);
  cursor: pointer;
  transition: all 0.15s;
}
.port-remove-btn:hover {
  color: var(--color-error, #dc2626);
  background: var(--color-error-bg, rgba(239, 68, 68, 0.06));
}

/* ── If condition list ────────────────────────────────── */

.logic-toggle {
  display: flex;
  gap: 4px;
}

.logic-toggle__btn {
  flex: 1;
  padding: 6px 10px;
  font-size: 12px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.1));
  background: none;
  color: var(--text-secondary-color, #57534e);
  cursor: pointer;
  transition: all 0.15s;
}

.logic-toggle__btn--active {
  border-color: var(--theme-primary, #5a8f4e);
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 6%, transparent);
  color: var(--theme-primary, #5a8f4e);
  font-weight: 500;
}

.if-conditions {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.if-condition-row {
  display: flex;
  align-items: center;
  gap: 4px;
}

.if-condition-row__index {
  width: 18px;
  font-size: 10px;
  color: var(--text-tertiary-color, #a8a29e);
  text-align: center;
  flex-shrink: 0;
}

.if-condition-row__input-wrap {
  flex: 1;
  position: relative;
  min-width: 0;
}

.if-condition-row__input {
  width: 100%;
  padding-right: 28px !important;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 11px;
}

.if-condition-row__operator {
  width: 88px;
  flex-shrink: 0;
  font-size: 11px;
  padding: 5px 4px;
}

.var-insert-btn--small {
  width: 22px;
  height: 22px;
  font-size: 9px;
  line-height: 22px;
  padding: 0;
}

.validation-error {
  font-size: 11px;
  color: var(--color-error, #dc2626);
  margin-top: 4px;
}

.btn {
  padding: 6px 14px;
  font-size: 13px;
  border-radius: var(--radius-xs, 4px);
  cursor: pointer;
  transition: all 0.15s;
  border: none;
}

.btn--ghost {
  background: var(--surface-tertiary-color, #e8e4de);
  color: var(--text-secondary-color, #57534e);
}
.btn--ghost:hover {
  background: var(--surface-hover-color, #e2ddd7);
}

.btn--primary {
  background: var(--theme-primary, #5a8f4e);
  color: white;
}
.btn--primary:hover {
  opacity: 0.9;
}

.btn--danger {
  background: var(--color-error-bg, rgba(239, 68, 68, 0.06));
  color: var(--color-error, #dc2626);
}
.btn--danger:hover {
  background: var(--color-error-bg, rgba(239, 68, 68, 0.1));
}

/* 变量插入按钮 */
.form-input-with-action {
  position: relative;
}

.var-insert-btn {
  position: absolute;
  top: 6px;
  right: 6px;
  width: 28px;
  height: 28px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.08));
  background: var(--strong-background-color, #fff);
  color: var(--text-tertiary-color, #a8a29e);
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 10px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
  z-index: 1;
}

.var-insert-btn:hover {
  border-color: var(--theme-primary, #5a8f4e);
  color: var(--theme-primary, #5a8f4e);
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 6%, transparent);
}

/* Switch 分支编辑 */
.switch-cases {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.switch-case-row {
  display: grid;
  grid-template-columns: 1fr 1fr 32px;
  gap: 6px;
  align-items: center;
}
</style>
