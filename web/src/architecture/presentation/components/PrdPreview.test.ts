import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { describe, expect, it } from 'vitest'
import PrdPreview from './PrdPreview.vue'

const SlotStub = defineComponent({
  template: '<div><slot /></div>',
})

const ChartPreviewStub = defineComponent({
  name: 'PrdChartPreview',
  props: {
    previewData: {
      type: Array,
      default: () => [],
    },
  },
  template: '<div class="chart-preview-stub">chart</div>',
})

const ButtonStub = defineComponent({
  emits: ['click'],
  template: '<button type="button" @click="$emit(\'click\', $event)"><slot /></button>',
})

function globalStubs() {
  return {
    PrdChartPreview: ChartPreviewStub,
    TableIcon: SlotStub,
    FormIcon: SlotStub,
    ChartIcon: SlotStub,
    ElIcon: SlotStub,
    ElTag: SlotStub,
    ElButton: ButtonStub,
    ElEmpty: SlotStub,
  }
}

function baseData(): any {
  return {
    kind: 'agent_app_prd',
    project: {
      name: '工单管理',
      code: 'ticket_management',
      create_new_directory: true,
    },
    models: [
      {
        name: 'Ticket',
        display_name: '工单',
        table_name: 'ticket',
        fields: [
          {
            display_name: '工单标题',
            json_name: 'title',
            widget: 'name:工单标题;type:input',
          },
          {
            display_name: '工单状态',
            json_name: 'status',
            widget: 'name:工单状态;type:select;options:待处理,处理中,已完成',
            validate: 'required',
          },
        ],
      },
    ],
    functions: [
      {
        title: '工单列表',
        type: 'table',
        route: 'ticket_list.table',
        model: 'Ticket',
        table: {
          request_fields: [
            {
              field: '状态',
              type: '下拉选择',
              required: false,
              default: '全部',
              description: 'options:待处理,处理中,已完成',
            },
          ],
          columns: ['工单标题', '工单状态'],
          sample_rows: [
            {
              工单标题: '打印机无法连接',
              工单状态: '待处理',
            },
          ],
        },
      },
      {
        title: '创建工单',
        type: 'form',
        route: 'ticket_create.form',
        model: 'Ticket',
        form: {},
      },
      {
        title: '工单统计',
        type: 'chart',
        route: 'ticket_stats.chart',
        model: 'Ticket',
        chart: {
          chart_type: 'pie',
          dimension: '工单状态',
          metrics: ['工单数量'],
          filters: [
            { name: '开始时间', type: 'datetime', required: false, desc: '统计开始时间' },
            { name: '结束时间', type: 'datetime', required: false, desc: '统计结束时间' },
          ],
          preview_data: [
            { name: '待处理', value: 8 },
            { name: '处理中', value: 5 },
            { name: '已完成', value: 21 },
          ],
          summary: [
            { name: '总工单数', value: 34, desc: '当前筛选条件下的工单总数' },
          ],
        },
      },
    ],
    confirmation: {
      question: '请确认是否按以上 PRD 创建目录和生成代码。',
    },
  }
}

function mountPreview(data: any = baseData()) {
  return mount(PrdPreview, {
    props: { data },
    global: {
      stubs: globalStubs(),
    },
  })
}

describe('PrdPreview', () => {
  it('switches function previews by thumbnail index', async () => {
    const wrapper = mountPreview()

    expect(wrapper.find('.prd-stage-name').text()).toBe('工单列表')
    expect(wrapper.find('.prd-basic-table').exists()).toBe(true)

    const thumbs = wrapper.findAll('.prd-slide-thumb')
    expect(thumbs).toHaveLength(3)
    expect(thumbs[0]!.classes()).toContain('is-active')

    await thumbs[2]!.trigger('pointerup')
    await nextTick()
    expect(wrapper.find('.prd-stage-name').text()).toBe('工单统计')
    expect(wrapper.find('.chart-preview-stub').exists()).toBe(true)
    expect(wrapper.findAll('.prd-slide-thumb')[2]!.classes()).toContain('is-active')

    await wrapper.findAll('.prd-slide-thumb')[1]!.trigger('pointerup')
    await nextTick()
    expect(wrapper.find('.prd-stage-name').text()).toBe('创建工单')
    expect(wrapper.text()).toContain('请求字段')
    expect(wrapper.text()).toContain('工单状态')
  })

  it('renders table request fields, columns and sample rows without runtime widgets', () => {
    const wrapper = mountPreview()

    expect(wrapper.text()).toContain('搜索条件')
    expect(wrapper.text()).toContain('全部')
    expect(wrapper.text()).toContain('待处理')
    expect(wrapper.find('.prd-basic-table').text()).toContain('打印机无法连接')
    expect(wrapper.findComponent(ChartPreviewStub).exists()).toBe(false)
  })

  it('sorts function previews by explicit order', async () => {
    const data = baseData()
    data.functions = [
      { ...data.functions[2], order: 3 },
      { ...data.functions[1], order: 2 },
      { ...data.functions[0], order: 1 },
    ]
    const wrapper = mountPreview(data)

    expect(wrapper.find('.prd-stage-name').text()).toBe('工单列表')
    expect(wrapper.find('.prd-step-badge').text()).toBe('01')
    const thumbs = wrapper.findAll('.prd-slide-thumb')
    expect(thumbs.map(item => item.find('.prd-slide-step').text())).toEqual(['01', '02', '03'])

    await thumbs[2]!.trigger('pointerup')
    await nextTick()
    expect(wrapper.find('.prd-stage-name').text()).toBe('工单统计')
    expect(wrapper.find('.prd-step-badge').text()).toBe('03')
  })

  it('renders table columns directly when no model exists', () => {
    const wrapper = mountPreview({
      kind: 'agent_app_prd',
      project: {
        name: '收银台',
        code: 'cashier',
        create_new_directory: true,
      },
      functions: [
        {
          title: '支付记录表',
          type: 'table',
          route: 'cashier_payment_record_list.table',
          table: {
            readonly: false,
            columns: ['订单号', '会员姓名', '实付金额', '状态'],
            sample_rows: [
              {
                订单号: 'ORD202501200001',
                会员姓名: '张三',
                实付金额: '13.50',
                状态: '支付成功',
              },
            ],
          },
        },
      ],
    })

    expect(wrapper.find('.prd-basic-table').text()).toContain('ORD202501200001')
    expect(wrapper.find('.prd-basic-table').text()).toContain('支付成功')
  })

  it('renders model options in lightweight form preview', () => {
    const wrapper = mountPreview({
      kind: 'agent_app_prd',
      project: {
        name: '工单管理',
        code: 'ticket_management',
        create_new_directory: true,
      },
      models: [
        {
          name: 'Ticket',
          fields: [
            {
              name: '工单状态',
              widget: 'name:工单状态;type:select',
              validate: 'required',
              options: ['待处理', '处理中', '已完成'],
              render_default: '待处理',
            },
          ],
        },
      ],
      functions: [
        {
          title: '创建工单',
          type: 'form',
          route: 'ticket_create.form',
          model: 'Ticket',
          form: {},
        },
      ],
    })

    expect(wrapper.text()).toContain('工单状态')
    expect(wrapper.text()).toContain('必填')
    expect(wrapper.text()).toContain('待处理')
    expect(wrapper.text()).toContain('处理中')
    expect(wrapper.text()).toContain('已完成')
  })

  it('passes lightweight chart preview data and renders summary', async () => {
    const wrapper = mountPreview()
    await wrapper.findAll('.prd-slide-thumb')[2]!.trigger('pointerup')
    await nextTick()

    const chartPreview = wrapper.findComponent(ChartPreviewStub)
    expect(chartPreview.props('previewData')).toEqual([
      { name: '待处理', value: 8 },
      { name: '处理中', value: 5 },
      { name: '已完成', value: 21 },
    ])
    expect(wrapper.text()).toContain('开始时间')
    expect(wrapper.text()).toContain('结束时间')
    expect(wrapper.text()).toContain('总工单数')
    expect(wrapper.text()).toContain('34')
  })

  it('emits confirm payload with remark', async () => {
    const wrapper = mountPreview()

    await wrapper.find('textarea').setValue('按这个做')
    const confirmButton = wrapper.findAll('button').find(button => button.text().includes('确认 PRD'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')

    expect(wrapper.emitted('confirm')?.[0]?.[0]).toMatchObject({
      remark: '按这个做',
    })
  })
})
