<template>
  <span
    :class="['agent-employee-mascot', `is-${variant}`, `is-${state}`]"
    :data-agent-state="state"
    :data-agent-variant="variant"
    role="img"
    :aria-label="label || defaultLabel"
  >
    <svg
      v-if="variant === 'mark'"
      class="agent-employee-art"
      viewBox="0 0 48 48"
      aria-hidden="true"
      focusable="false"
    >
      <circle class="agent-state-halo" cx="24" cy="24" r="21" />
      <g class="agent-character">
        <path
          class="agent-hood"
          d="M8.4 24.8C8.4 13.1 15.6 5.5 25.1 5.5c9.3 0 15.8 7.1 15.8 18.9 0 8.7-5.2 15.9-16.5 15.9-10.8 0-16-6.8-16-15.5Z"
        />
        <path class="agent-hood-highlight" d="M17.2 8.6c4.9-3 10.8-3.5 15.6-.8-5.5.6-9.5 2.4-12.2 5.7-2.3 2.8-3.2 6.4-3.1 10.4-2.2-5.4-2-11 .3-15.3Z" />
        <ellipse class="agent-face" cx="24.1" cy="25.1" rx="11.4" ry="10.4" />
        <path class="agent-hair" d="M14.9 21.7c1.3-6.7 5.4-10.2 11.3-10.2 4.7 0 8.6 2.2 10.4 6.1-4.6-.1-7.6-1.2-10-3.7-2.5 3.8-6.5 6.4-11.7 7.8Z" />
        <g v-if="state === 'paused'" class="agent-eyes agent-eyes-closed">
          <path d="M18.2 25.9c1.3 1.2 2.7 1.2 4 0" />
          <path d="M26.8 25.9c1.3 1.2 2.7 1.2 4 0" />
        </g>
        <g v-else class="agent-eyes">
          <ellipse cx="20.2" cy="25.3" rx="1.45" ry="2.2" />
          <ellipse cx="28.8" cy="25.3" rx="1.45" ry="2.2" />
        </g>
        <path
          v-if="state === 'failed'"
          class="agent-mouth agent-mouth-alert"
          d="M23 30.8c.6-.8 1.5-.8 2.1 0"
        />
        <path v-else class="agent-mouth" d="M21.1 29.7c1.9 2 4 2 5.9 0" />
        <g class="agent-earpiece">
          <circle cx="37.1" cy="23.8" r="4.3" />
          <circle class="agent-earpiece-light" cx="37.1" cy="23.8" r="2.1" />
        </g>
        <circle class="agent-state-light" cx="35.9" cy="36.5" r="3" />
        <path v-if="state === 'failed'" class="agent-alert-mark" d="M35.9 34.7v2.1m0 1.5v.1" />
      </g>
      <g v-if="state === 'working'" class="agent-data-sparks">
        <circle cx="7" cy="15" r="1.2" />
        <circle cx="5.7" cy="22" r=".85" />
        <circle cx="9.3" cy="9.6" r=".72" />
      </g>
      <g v-if="state === 'paused'" class="agent-sleep-sign">
        <path d="M34 7.8h5l-5 5h5" />
      </g>
    </svg>

    <svg
      v-else
      class="agent-employee-art"
      viewBox="0 0 88 76"
      aria-hidden="true"
      focusable="false"
    >
      <ellipse class="agent-platform-shadow" cx="43.5" cy="69.5" rx="31" ry="4.5" />
      <g class="agent-character">
        <g class="agent-head">
          <path
            class="agent-hood"
            d="M22 27.5C22 13.2 30.6 4.8 42.3 4.8c11.4 0 20 8.6 20 22.6 0 11.7-7.2 20.2-20.4 20.2C29 47.6 22 39.7 22 27.5Z"
          />
          <path class="agent-hood-highlight" d="M31.4 8.1c5.5-3.5 12.1-3.9 17.6-.8-6.4.7-10.8 2.7-13.7 6.7-2.6 3.4-3.5 7.8-3.2 12.5-2.8-6.2-2.8-12.7-.7-18.4Z" />
          <ellipse class="agent-face" cx="42.4" cy="28.4" rx="13.8" ry="12.3" />
          <path class="agent-hair" d="M30.3 24.2c1.7-8 6.4-12.1 13.4-12.1 5.6 0 10.2 2.5 12.5 7.1-5.4-.1-9-1.4-11.9-4.4-3 4.5-7.8 7.5-14 9.4Z" />
          <g v-if="state === 'paused'" class="agent-eyes agent-eyes-closed">
            <path d="M35.1 29.1c1.7 1.4 3.3 1.4 5 0" />
            <path d="M45.7 29.1c1.7 1.4 3.3 1.4 5 0" />
          </g>
          <g v-else class="agent-eyes">
            <ellipse cx="37.8" cy="28.4" rx="1.75" ry="2.65" />
            <ellipse cx="48.2" cy="28.4" rx="1.75" ry="2.65" />
          </g>
          <g v-if="state === 'failed'" class="agent-brows">
            <path d="m34.8 23.5 4.8 1.2" />
            <path d="m50.9 23.5-4.8 1.2" />
          </g>
          <path
            v-if="state === 'failed'"
            class="agent-mouth agent-mouth-alert"
            d="M41.1 34.5c1.1-1.2 2.4-1.2 3.5 0"
          />
          <path v-else class="agent-mouth" d="M38.7 33.8c2.5 2.3 5 2.3 7.5 0" />
          <g class="agent-earpiece">
            <circle cx="58.5" cy="26.8" r="5.1" />
            <circle class="agent-earpiece-light" cx="58.5" cy="26.8" r="2.45" />
          </g>
        </g>

        <path class="agent-body" d="M28.3 58.8c.5-10.8 5.4-16.7 14.6-16.7 9.3 0 14 5.9 14.5 16.7Z" />
        <path class="agent-collar-light" d="M39.6 45.2h6.6c.9 0 1.5.7 1.2 1.5l-.4 1.1c-.2.6-.8 1-1.5 1h-5.2c-.7 0-1.3-.4-1.5-1l-.4-1.1c-.3-.8.3-1.5 1.2-1.5Z" />
        <g class="agent-badge">
          <circle cx="52" cy="50.2" r="4.2" />
          <path d="M50.3 47.9v4.7m.1-2.3 3-2.4m-3 2.4 3 2.5" />
        </g>

        <g class="agent-arm agent-arm-left">
          <path d="M31.8 50.2c-4.8 1.6-7.3 4.4-8.1 8.4l5.4 1.5c1-2.7 2.7-4.2 5.4-4.8Z" />
          <circle class="agent-hand" cx="28.1" cy="59.5" r="3.1" />
        </g>
        <g class="agent-arm agent-arm-right">
          <path d="M54 50.2c4.8 1.6 7.3 4.4 8.1 8.4l-5.4 1.5c-1-2.7-2.7-4.2-5.4-4.8Z" />
          <circle class="agent-hand" cx="57.7" cy="59.5" r="3.1" />
        </g>

        <g class="agent-console">
          <path class="agent-console-screen" d="M19.5 50.3h46.9l-4.1 15H23.6Z" />
          <path class="agent-console-edge" d="M22.6 64.7h40.7c1.4 0 2.5 1.1 2.5 2.5H20.1c0-1.4 1.1-2.5 2.5-2.5Z" />
          <path class="agent-console-glow" d="M38.1 58.7h9.7" />
          <circle class="agent-console-dot" cx="58.4" cy="58.8" r="1.1" />
        </g>
      </g>

      <g v-if="state === 'working'" class="agent-data-sparks">
        <path d="M13.5 37.5h5.4m-7.3 4.4h7.3m50-8.3h5.5m-5.5 4.4h8.1" />
        <circle cx="14.2" cy="31.8" r="1.25" />
        <circle cx="72.8" cy="28.9" r="1" />
      </g>
      <g v-if="state === 'ready'" class="agent-ready-signal">
        <path d="M67.5 16.6c4.3 1.5 7.4 5.2 8.1 9.7M68.6 21c2.1 1.1 3.5 3.1 3.7 5.5" />
        <circle cx="68.8" cy="27.8" r="1.4" />
      </g>
      <g v-if="state === 'paused'" class="agent-sleep-sign">
        <path d="M63.6 11.2h7.3l-7.3 7h7.3M70.6 5.8h5l-5 4.8h5" />
      </g>
      <g v-if="state === 'failed'" class="agent-alert-sign">
        <path d="M70 10.7 80.5 29H59.4Z" />
        <path class="agent-alert-mark" d="M70 16.5v6.4m0 2.6v.2" />
      </g>
    </svg>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type AgentEmployeeState = 'working' | 'ready' | 'paused' | 'failed'
type AgentEmployeeVariant = 'mark' | 'employee'

const props = withDefaults(defineProps<{
  state?: AgentEmployeeState
  variant?: AgentEmployeeVariant
  label?: string
}>(), {
  state: 'ready',
  variant: 'employee',
  label: '',
})

const stateLabels: Record<AgentEmployeeState, string> = {
  working: '智能员工正在处理',
  ready: '智能员工正在待命',
  paused: '智能员工已暂停',
  failed: '智能员工需要关注',
}

const defaultLabel = computed(() => stateLabels[props.state])
</script>

<style scoped lang="scss">
.agent-employee-mascot {
  --agent-hood: #172554;
  --agent-hood-highlight: #2b3f7f;
  --agent-face: #ffd8aa;
  --agent-ink: #111c3f;
  --agent-cyan: #4de2ff;
  --agent-state: #10b981;
  --agent-state-soft: rgba(16, 185, 129, 0.18);

  position: relative;
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  overflow: visible;
  line-height: 1;
  vertical-align: middle;
}

.agent-employee-mascot.is-mark {
  width: 22px;
  height: 22px;
}

.agent-employee-mascot.is-employee {
  width: 58px;
  height: 52px;
}

.agent-employee-mascot.is-working {
  --agent-state: #3b82f6;
  --agent-state-soft: rgba(59, 130, 246, 0.2);
}

.agent-employee-mascot.is-paused {
  --agent-state: #94a3b8;
  --agent-state-soft: rgba(148, 163, 184, 0.18);
  filter: saturate(0.72);
}

.agent-employee-mascot.is-failed {
  --agent-state: #f97316;
  --agent-state-soft: rgba(249, 115, 22, 0.2);
}

.agent-employee-art {
  display: block;
  width: 100%;
  height: 100%;
  overflow: visible;
}

.agent-state-halo {
  fill: var(--agent-state-soft);
  stroke: var(--agent-state);
  stroke-opacity: 0.24;
  stroke-width: 1.5;
}

.agent-hood,
.agent-body,
.agent-arm {
  fill: var(--agent-hood);
}

.agent-hood-highlight {
  fill: var(--agent-hood-highlight);
  opacity: 0.88;
}

.agent-face,
.agent-hand {
  fill: var(--agent-face);
}

.agent-hair {
  fill: #111b46;
}

.agent-eyes {
  fill: var(--agent-ink);
  stroke: var(--agent-ink);
  stroke-linecap: round;
  stroke-width: 1.25;
  transform-box: fill-box;
  transform-origin: center;
}

.agent-eyes-closed {
  fill: none;
}

.agent-mouth {
  fill: none;
  stroke: #27345d;
  stroke-linecap: round;
  stroke-width: 1.35;
}

.agent-mouth-alert {
  stroke: #9a3412;
  stroke-width: 1.7;
}

.agent-brows {
  fill: none;
  stroke: #713f12;
  stroke-linecap: round;
  stroke-width: 1.4;
}

.agent-earpiece {
  fill: #243b78;
  stroke: #3458a6;
  stroke-width: 1;
}

.agent-earpiece-light,
.agent-collar-light {
  fill: var(--agent-cyan);
  filter: drop-shadow(0 0 2px rgba(77, 226, 255, 0.8));
}

.agent-state-light {
  fill: var(--agent-state);
  stroke: #fff;
  stroke-width: 1.2;
  filter: drop-shadow(0 0 2px var(--agent-state));
}

.agent-badge circle {
  fill: #1e3a5f;
  stroke: var(--agent-cyan);
  stroke-opacity: 0.65;
  stroke-width: 0.8;
}

.agent-badge path {
  fill: none;
  stroke: #b9f4ff;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 1.15;
}

.agent-console-screen {
  fill: #203766;
  stroke: #62dff9;
  stroke-opacity: 0.68;
  stroke-width: 0.9;
}

.agent-console-edge {
  fill: #14254b;
  stroke: #395b96;
  stroke-width: 0.8;
}

.agent-console-glow {
  fill: none;
  stroke: var(--agent-state);
  stroke-linecap: round;
  stroke-width: 1.8;
  filter: drop-shadow(0 0 2px var(--agent-state));
}

.agent-console-dot {
  fill: var(--agent-cyan);
}

.agent-platform-shadow {
  fill: rgba(15, 23, 42, 0.15);
}

.agent-data-sparks,
.agent-ready-signal,
.agent-sleep-sign {
  fill: none;
  stroke: var(--agent-state);
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 1.6;
}

.agent-alert-sign {
  fill: rgba(249, 115, 22, 0.12);
  stroke: #f97316;
  stroke-linejoin: round;
  stroke-width: 1.6;
  transform-box: fill-box;
  transform-origin: center;
}

.agent-alert-mark {
  fill: none;
  stroke: #f97316;
  stroke-linecap: round;
  stroke-width: 1.8;
}

.is-working .agent-character {
  animation: agent-focus-bob 1s ease-in-out infinite alternate;
}

.is-working .agent-arm-left {
  animation: agent-type-left 0.34s ease-in-out infinite alternate;
  transform-box: fill-box;
  transform-origin: 78% 20%;
}

.is-working .agent-arm-right {
  animation: agent-type-right 0.34s 0.17s ease-in-out infinite alternate;
  transform-box: fill-box;
  transform-origin: 22% 20%;
}

.is-working .agent-console-glow,
.is-working .agent-state-light,
.is-working .agent-earpiece-light {
  animation: agent-console-pulse 0.9s ease-in-out infinite alternate;
}

.is-working .agent-data-sparks {
  animation: agent-data-flow 1.15s ease-in-out infinite;
}

.is-ready .agent-eyes {
  animation: agent-blink 5.2s ease-in-out infinite;
}

.is-ready.is-employee .agent-arm-right {
  animation: agent-ready-wave 2.8s ease-in-out infinite;
  transform-box: fill-box;
  transform-origin: 22% 20%;
}

.is-ready .agent-ready-signal {
  animation: agent-signal 2.2s ease-out infinite;
  transform-box: fill-box;
  transform-origin: left bottom;
}

.is-paused .agent-head {
  transform: translate(1px, 2px) rotate(4deg);
  transform-box: fill-box;
  transform-origin: center bottom;
}

.is-paused .agent-sleep-sign {
  animation: agent-sleep 2.4s ease-in-out infinite;
}

.is-failed .agent-alert-sign,
.is-failed .agent-state-light {
  animation: agent-alert-pulse 1.35s ease-in-out infinite;
}

.is-failed .agent-character {
  animation: agent-alert-nudge 3s ease-in-out infinite;
}

@keyframes agent-focus-bob {
  from { transform: translateY(0); }
  to { transform: translateY(-1.5px); }
}

@keyframes agent-type-left {
  from { transform: translateY(0) rotate(0); }
  to { transform: translateY(1.4px) rotate(4deg); }
}

@keyframes agent-type-right {
  from { transform: translateY(1.4px) rotate(-4deg); }
  to { transform: translateY(0) rotate(0); }
}

@keyframes agent-console-pulse {
  from { opacity: 0.58; }
  to { opacity: 1; }
}

@keyframes agent-data-flow {
  0% { opacity: 0; transform: translateY(2px); }
  45% { opacity: 1; }
  100% { opacity: 0; transform: translateY(-3px); }
}

@keyframes agent-blink {
  0%,
  44%,
  48%,
  100% { transform: scaleY(1); }
  46% { transform: scaleY(0.08); }
}

@keyframes agent-ready-wave {
  0%,
  78%,
  100% { transform: rotate(0); }
  84% { transform: rotate(-7deg) translateY(-1px); }
  91% { transform: rotate(5deg); }
}

@keyframes agent-signal {
  0% { opacity: 0; transform: scale(0.8); }
  35% { opacity: 0.9; }
  100% { opacity: 0; transform: scale(1.08); }
}

@keyframes agent-sleep {
  0% { opacity: 0; transform: translate(-1px, 2px); }
  45% { opacity: 0.9; }
  100% { opacity: 0; transform: translate(2px, -3px); }
}

@keyframes agent-alert-pulse {
  0%,
  100% { opacity: 0.72; transform: scale(0.94); }
  50% { opacity: 1; transform: scale(1.04); }
}

@keyframes agent-alert-nudge {
  0%,
  82%,
  100% { transform: translateX(0); }
  86% { transform: translateX(-1px); }
  90% { transform: translateX(1px); }
  94% { transform: translateX(-0.6px); }
}

@media (prefers-reduced-motion: reduce) {
  .agent-employee-mascot *,
  .agent-employee-mascot *::before,
  .agent-employee-mascot *::after {
    animation: none !important;
  }
}
</style>
