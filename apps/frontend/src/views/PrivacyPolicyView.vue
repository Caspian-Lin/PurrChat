<template>
  <div class="min-h-screen relative flex flex-col" style="background: var(--background-color)">
    <div class="absolute top-4 right-4 z-10">
      <ThemeSwitcher />
    </div>

    <div class="flex-1 w-full max-w-3xl mx-auto px-6 py-12">
      <!-- 返回按钮 -->
      <router-link
        to="/login"
        class="inline-flex items-center gap-1.5 text-sm mb-8 transition-colors"
        style="color: var(--text-secondary-color)"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="m15 18-6-6 6-6" />
        </svg>
        返回
      </router-link>

      <article class="privacy-content" style="color: var(--text-color)">
        <h1 class="font-display text-3xl font-bold mb-2" style="color: var(--theme-primary)">
          PurrChat 隐私政策
        </h1>
        <p class="text-sm mb-10" style="color: var(--text-tertiary-color)">
          最后更新日期：{{ updateDate }}
        </p>

        <section class="mb-8">
          <h2 class="text-xl font-semibold mb-3" style="color: var(--text-color)">引言</h2>
          <p class="leading-relaxed" style="color: var(--text-secondary-color)">
            PurrChat（以下简称"我们"）非常重视用户的隐私保护。本隐私政策旨在向您说明我们如何收集、使用、存储和保护您的个人信息。请您在使用我们的服务前，仔细阅读本隐私政策。
          </p>
        </section>

        <section class="mb-8">
          <h2 class="text-xl font-semibold mb-3" style="color: var(--text-color)">
            一、我们收集的信息
          </h2>
          <p class="mb-2 leading-relaxed" style="color: var(--text-secondary-color)">
            为了向您提供服务，我们可能会收集以下类型的信息：
          </p>
          <ul
            class="list-disc pl-6 space-y-1.5 leading-relaxed"
            style="color: var(--text-secondary-color)"
          >
            <li><strong>账户信息：</strong>用户名、邮箱地址、手机号码（您在注册时提供）</li>
            <li><strong>聊天记录：</strong>您与其他用户或 AI 助手之间的对话内容</li>
            <li><strong>设备信息：</strong>设备型号、操作系统版本、浏览器类型等</li>
            <li><strong>日志信息：</strong>访问时间、IP 地址、操作记录等</li>
          </ul>
        </section>

        <section class="mb-8">
          <h2 class="text-xl font-semibold mb-3" style="color: var(--text-color)">
            二、信息的使用
          </h2>
          <p class="mb-2 leading-relaxed" style="color: var(--text-secondary-color)">
            我们收集的信息将用于以下目的：
          </p>
          <ul
            class="list-disc pl-6 space-y-1.5 leading-relaxed"
            style="color: var(--text-secondary-color)"
          >
            <li>提供、维护和改进我们的服务</li>
            <li>处理您的请求并与您进行沟通</li>
            <li>保障账户安全和防范欺诈行为</li>
            <li>分析服务使用情况以优化用户体验</li>
            <li>遵守法律法规的要求</li>
          </ul>
        </section>

        <section class="mb-8">
          <h2 class="text-xl font-semibold mb-3" style="color: var(--text-color)">
            三、信息的存储与保护
          </h2>
          <p class="leading-relaxed" style="color: var(--text-secondary-color)">
            我们采用行业标准的安全措施来保护您的个人信息，包括数据加密传输（TLS/SSL）、访问权限控制和定期安全审计。您的数据存储在安全的服务器环境中，我们会在实现收集目的所必需的合理期限内保留您的个人信息。
          </p>
        </section>

        <section class="mb-8">
          <h2 class="text-xl font-semibold mb-3" style="color: var(--text-color)">
            四、信息的共享与披露
          </h2>
          <p class="leading-relaxed" style="color: var(--text-secondary-color)">
            我们不会将您的个人信息出售给第三方。仅在以下情况下，我们可能会共享您的信息：获得您的明确同意；为遵守法律法规的要求；为保护我们、其他用户或公众的权利、财产或安全。
          </p>
        </section>

        <section class="mb-8">
          <h2 class="text-xl font-semibold mb-3" style="color: var(--text-color)">五、您的权利</h2>
          <p class="mb-2 leading-relaxed" style="color: var(--text-secondary-color)">
            您对您的个人信息享有以下权利：
          </p>
          <ul
            class="list-disc pl-6 space-y-1.5 leading-relaxed"
            style="color: var(--text-secondary-color)"
          >
            <li><strong>访问权：</strong>您有权查看您的个人信息</li>
            <li><strong>更正权：</strong>您有权要求更正不准确的信息</li>
            <li><strong>删除权：</strong>您有权要求删除您的个人信息</li>
            <li><strong>撤回同意：</strong>您有权撤回之前给予的同意</li>
          </ul>
        </section>

        <section class="mb-8">
          <h2 class="text-xl font-semibold mb-3" style="color: var(--text-color)">
            六、Cookie 的使用
          </h2>
          <p class="leading-relaxed" style="color: var(--text-secondary-color)">
            我们使用 Cookie 和类似技术来提升您的使用体验。您可以通过浏览器设置管理 Cookie
            偏好，但禁用 Cookie 可能会影响部分功能的正常使用。
          </p>
        </section>

        <section class="mb-8">
          <h2 class="text-xl font-semibold mb-3" style="color: var(--text-color)">
            七、未成年人保护
          </h2>
          <p class="leading-relaxed" style="color: var(--text-secondary-color)">
            我们的服务面向一般成年用户。如果您是未满 18
            周岁的未成年人，请在监护人的指导下使用本服务，并在监护人同意后提供个人信息。
          </p>
        </section>

        <section class="mb-8">
          <h2 class="text-xl font-semibold mb-3" style="color: var(--text-color)">
            八、政策的更新
          </h2>
          <p class="leading-relaxed" style="color: var(--text-secondary-color)">
            我们可能会不时更新本隐私政策。更新后的政策将在本页面发布，并修改"最后更新日期"。我们建议您定期查看本政策以了解任何变更。
          </p>
        </section>

        <section class="mb-8">
          <h2 class="text-xl font-semibold mb-3" style="color: var(--text-color)">九、联系我们</h2>
          <p class="leading-relaxed" style="color: var(--text-secondary-color)">
            如果您对本隐私政策有任何疑问或建议，请通过以下方式联系我们：
          </p>
          <p class="mt-2 leading-relaxed" style="color: var(--text-secondary-color)">
            邮箱：<a
              href="mailto:support@purrchat.com"
              class="underline"
              style="color: var(--theme-primary)"
              >support@purrchat.com</a
            >
          </p>
        </section>
      </article>
    </div>

    <!-- 底部：ICP 备案 -->
    <footer
      class="w-full py-6 text-center text-xs border-t"
      style="color: var(--text-tertiary-color); border-color: var(--border-subtle-color)"
    >
      <div class="space-y-1">
        <p>
          <a href="/privacy" class="transition-colors" style="color: var(--text-secondary-color)">
            隐私政策
          </a>
        </p>
        <p>© {{ currentYear }} PurrChat. All rights reserved.</p>
        <!-- ICP 备案信息 —— 请在获得备案号后替换下方内容 -->
        <p class="icp-info">
          <a
            href="https://beian.miit.gov.cn/"
            target="_blank"
            rel="noopener noreferrer"
            class="transition-colors"
            style="color: var(--text-tertiary-color)"
          >
            京ICP备XXXXXXXX号
          </a>
        </p>
        <!-- 公安备案信息 —— 请在获得备案号后替换下方内容 -->
        <p class="police-info">
          <a
            href="http://www.beian.gov.cn/"
            target="_blank"
            rel="noopener noreferrer"
            class="transition-colors"
            style="color: var(--text-tertiary-color)"
          >
            京公网安备 XXXXXXXXXXXXXXXX 号
          </a>
        </p>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import ThemeSwitcher from '../components/ThemeSwitcher.vue';

const now = new Date();
const updateDate = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
const currentYear = now.getFullYear();
</script>

<style scoped>
.privacy-content h2 {
  position: relative;
  padding-bottom: 8px;
}

.privacy-content h2::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  width: 32px;
  height: 2px;
  border-radius: var(--radius-full);
  background: var(--theme-primary);
  opacity: 0.5;
}

.privacy-content a {
  text-decoration: none;
}

.privacy-content a:hover {
  opacity: 0.8;
}
</style>
