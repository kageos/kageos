export type LocaleCode =
  | "en"
  | "zh-CN"
  | "es"
  | "fr"
  | "de"
  | "ja"
  | "ko"
  | "pt-BR"
  | "ar";

export type LocaleConfig = {
  code: LocaleCode;
  label: string;
  nativeLabel: string;
  dir: "ltr" | "rtl";
};

export const defaultLocale: LocaleCode = "en";

export const locales: LocaleConfig[] = [
  { code: "en", label: "English", nativeLabel: "English", dir: "ltr" },
  { code: "zh-CN", label: "Chinese", nativeLabel: "简体中文", dir: "ltr" },
  { code: "es", label: "Spanish", nativeLabel: "Español", dir: "ltr" },
  { code: "fr", label: "French", nativeLabel: "Français", dir: "ltr" },
  { code: "de", label: "German", nativeLabel: "Deutsch", dir: "ltr" },
  { code: "ja", label: "Japanese", nativeLabel: "日本語", dir: "ltr" },
  { code: "ko", label: "Korean", nativeLabel: "한국어", dir: "ltr" },
  { code: "pt-BR", label: "Portuguese", nativeLabel: "Português", dir: "ltr" },
  { code: "ar", label: "Arabic", nativeLabel: "العربية", dir: "rtl" },
];

export type TranslationKey =
  | "brand.subtitle"
  | "nav.templates"
  | "nav.docs"
  | "nav.connectHub"
  | "locale.label"
  | "badge.officialTemplates"
  | "badge.bundleReady"
  | "hero.title"
  | "hero.description"
  | "hero.explore"
  | "hero.viewStructure"
  | "featured.label"
  | "featured.empty"
  | "catalog.title"
  | "catalog.description"
  | "catalog.searchPlaceholder"
  | "catalog.allCategories"
  | "catalog.all"
  | "metric.functions"
  | "metric.docs"
  | "metric.official"
  | "common.yes"
  | "common.no"
  | "preview.title"
  | "preview.nodes"
  | "install.title"
  | "install.description"
  | "install.template"
  | "install.downloadKey"
  | "install.downloadKeyPlaceholder"
  | "install.targetPath"
  | "install.targetPathPlaceholder"
  | "install.submit"
  | "install.loading"
  | "install.status.idleTitle"
  | "install.status.errorTitle"
  | "install.status.successTitle"
  | "install.status.idleMessage"
  | "install.message.requestTicket"
  | "install.message.downloadBundle"
  | "install.message.installing"
  | "install.message.installed"
  | "install.message.bundleDownloaded"
  | "install.error.rejectedKey"
  | "install.error.noTicket"
  | "install.error.downloadFailed"
  | "install.error.installFailed"
  | "install.error.generic"
  | "empty.title"
  | "empty.description";

type TranslationMap = Record<TranslationKey, string>;

const en: TranslationMap = {
  "brand.subtitle": "Template Hub",
  "nav.templates": "Templates",
  "nav.docs": "Docs",
  "nav.connectHub": "Connect Hub",
  "locale.label": "Language",
  "badge.officialTemplates": "Official templates",
  "badge.bundleReady": "Capability bundle ready",
  "hero.title": "Launch useful business workspaces from polished templates.",
  "hero.description":
    "Browse paid and free templates, preview their function tree, then install the underlying capability bundle into your workspace.",
  "hero.explore": "Explore templates",
  "hero.viewStructure": "View bundle structure",
  "featured.label": "Featured template",
  "featured.empty": "No templates yet",
  "catalog.title": "Template catalog",
  "catalog.description":
    "Curated bundles for individuals, freelancers, and small teams.",
  "catalog.searchPlaceholder": "Search CRM, retail, tasks...",
  "catalog.allCategories": "All categories",
  "catalog.all": "All",
  "metric.functions": "Functions",
  "metric.docs": "Docs",
  "metric.official": "Official",
  "common.yes": "Yes",
  "common.no": "No",
  "preview.title": "Bundle structure",
  "preview.nodes": "{count} nodes",
  "install.title": "Install template",
  "install.description": "Download from Hub and install into a workspace path.",
  "install.template": "Template",
  "install.downloadKey": "Download key",
  "install.downloadKeyPlaceholder": "Paste one-time purchase key",
  "install.targetPath": "Target directory path",
  "install.targetPathPlaceholder": "/beiluo/work/openapi",
  "install.submit": "Fetch and install",
  "install.loading": "Installing...",
  "install.status.idleTitle": "Uses existing installer",
  "install.status.errorTitle": "Action needed",
  "install.status.successTitle": "Ready",
  "install.status.idleMessage":
    "The open-source app still owns installation. Hub only delivers the bundle JSON.",
  "install.message.requestTicket": "Requesting a one-time download ticket...",
  "install.message.downloadBundle": "Downloading capability bundle...",
  "install.message.installing": "Installing bundle into workspace...",
  "install.message.installed": "Template installed successfully.",
  "install.message.bundleDownloaded":
    "Bundle downloaded. Configure NEXT_PUBLIC_APP_SERVER_API_URL to install into {target}.",
  "install.error.rejectedKey": "Hub rejected the download key ({status})",
  "install.error.noTicket": "Hub did not return a ticket",
  "install.error.downloadFailed": "Bundle download failed ({status})",
  "install.error.installFailed": "Install request failed ({status})",
  "install.error.generic": "Install failed",
  "empty.title": "No templates available",
  "empty.description": "Start the Hub service or add products to the catalog.",
};

const dictionaries: Record<LocaleCode, Partial<TranslationMap>> = {
  en,
  "zh-CN": {
    "brand.subtitle": "模板中心",
    "nav.templates": "模板",
    "nav.docs": "文档",
    "nav.connectHub": "连接 Hub",
    "locale.label": "语言",
    "badge.officialTemplates": "官方模板",
    "badge.bundleReady": "能力包就绪",
    "hero.title": "用高质量模板快速启动真正可用的业务工作台。",
    "hero.description":
      "浏览免费和付费模板，预览函数与文档结构，然后把能力包安装到你的工作空间。",
    "hero.explore": "浏览模板",
    "hero.viewStructure": "查看包结构",
    "featured.label": "精选模板",
    "featured.empty": "暂无模板",
    "catalog.title": "模板目录",
    "catalog.description": "面向个人、自由职业者和小团队的精选能力包。",
    "catalog.searchPlaceholder": "搜索 CRM、零售、任务...",
    "catalog.allCategories": "全部分类",
    "catalog.all": "全部",
    "metric.functions": "函数",
    "metric.docs": "文档",
    "metric.official": "官方",
    "common.yes": "是",
    "common.no": "否",
    "preview.title": "包结构",
    "preview.nodes": "{count} 个节点",
    "install.title": "安装模板",
    "install.description": "从 Hub 下载并安装到工作空间目录。",
    "install.template": "模板",
    "install.downloadKey": "下载密钥",
    "install.downloadKeyPlaceholder": "粘贴一次性购买密钥",
    "install.targetPath": "目标目录路径",
    "install.submit": "获取并安装",
    "install.loading": "安装中...",
    "install.status.idleTitle": "使用现有安装器",
    "install.status.errorTitle": "需要处理",
    "install.status.successTitle": "已就绪",
    "install.status.idleMessage": "开源主项目负责安装，Hub 只交付 bundle JSON。",
    "install.message.requestTicket": "正在申请一次性下载票据...",
    "install.message.downloadBundle": "正在下载能力包...",
    "install.message.installing": "正在安装到工作空间...",
    "install.message.installed": "模板安装成功。",
    "install.message.bundleDownloaded":
      "能力包已下载。配置 NEXT_PUBLIC_APP_SERVER_API_URL 后即可安装到 {target}。",
    "install.error.rejectedKey": "Hub 拒绝了下载密钥（{status}）",
    "install.error.noTicket": "Hub 没有返回下载票据",
    "install.error.downloadFailed": "能力包下载失败（{status}）",
    "install.error.installFailed": "安装请求失败（{status}）",
    "install.error.generic": "安装失败",
    "empty.title": "暂无模板",
    "empty.description": "请先启动 Hub 服务，或在 catalog 中添加模板。",
  },
  es: {
    "brand.subtitle": "Hub de plantillas",
    "nav.templates": "Plantillas",
    "nav.docs": "Documentos",
    "nav.connectHub": "Conectar Hub",
    "locale.label": "Idioma",
    "badge.officialTemplates": "Plantillas oficiales",
    "badge.bundleReady": "Bundle listo",
    "hero.title": "Crea espacios de trabajo útiles desde plantillas pulidas.",
    "hero.description":
      "Explora plantillas gratuitas y de pago, revisa su árbol de funciones e instala el bundle en tu workspace.",
    "hero.explore": "Explorar plantillas",
    "hero.viewStructure": "Ver estructura",
    "featured.label": "Plantilla destacada",
    "featured.empty": "Aún no hay plantillas",
    "catalog.title": "Catálogo de plantillas",
    "catalog.description": "Bundles curados para personas, freelancers y equipos pequeños.",
    "catalog.searchPlaceholder": "Buscar CRM, retail, tareas...",
    "catalog.allCategories": "Todas las categorías",
    "catalog.all": "Todo",
    "metric.functions": "Funciones",
    "metric.docs": "Docs",
    "metric.official": "Oficial",
    "common.yes": "Sí",
    "common.no": "No",
    "preview.title": "Estructura del bundle",
    "preview.nodes": "{count} nodos",
    "install.title": "Instalar plantilla",
    "install.description": "Descarga desde Hub e instala en una ruta del workspace.",
    "install.template": "Plantilla",
    "install.downloadKey": "Clave de descarga",
    "install.downloadKeyPlaceholder": "Pega la clave de compra de un solo uso",
    "install.targetPath": "Ruta de destino",
    "install.submit": "Obtener e instalar",
    "install.loading": "Instalando...",
    "install.status.idleTitle": "Usa el instalador existente",
    "install.status.errorTitle": "Requiere acción",
    "install.status.successTitle": "Listo",
    "install.status.idleMessage": "La app open source instala; Hub solo entrega el bundle JSON.",
    "install.message.bundleDownloaded":
      "Bundle descargado. Configura NEXT_PUBLIC_APP_SERVER_API_URL para instalar en {target}.",
    "empty.title": "No hay plantillas disponibles",
    "empty.description": "Inicia Hub o agrega productos al catalog.",
  },
  fr: {
    "brand.subtitle": "Hub de modèles",
    "nav.templates": "Modèles",
    "nav.docs": "Docs",
    "nav.connectHub": "Connecter Hub",
    "locale.label": "Langue",
    "badge.officialTemplates": "Modèles officiels",
    "badge.bundleReady": "Bundle prêt",
    "hero.title":
      "Lancez des espaces de travail métier utiles à partir de modèles soignés.",
    "hero.description":
      "Parcourez les modèles gratuits et payants, prévisualisez leur arbre de fonctions, puis installez le bundle dans votre espace.",
    "hero.explore": "Explorer les modèles",
    "hero.viewStructure": "Voir la structure",
    "featured.label": "Modèle en vedette",
    "featured.empty": "Aucun modèle",
    "catalog.title": "Catalogue de modèles",
    "catalog.description": "Bundles sélectionnés pour individus, freelances et petites équipes.",
    "catalog.searchPlaceholder": "Rechercher CRM, retail, tâches...",
    "catalog.allCategories": "Toutes les catégories",
    "catalog.all": "Tout",
    "metric.functions": "Fonctions",
    "metric.docs": "Docs",
    "metric.official": "Officiel",
    "common.yes": "Oui",
    "common.no": "Non",
    "preview.title": "Structure du bundle",
    "preview.nodes": "{count} noeuds",
    "install.title": "Installer le modèle",
    "install.description": "Téléchargez depuis Hub et installez dans un chemin de workspace.",
    "install.template": "Modèle",
    "install.downloadKey": "Clé de téléchargement",
    "install.downloadKeyPlaceholder": "Collez la clé d'achat à usage unique",
    "install.targetPath": "Chemin de destination",
    "install.submit": "Récupérer et installer",
    "install.loading": "Installation...",
    "install.status.idleTitle": "Utilise l'installateur existant",
    "install.status.errorTitle": "Action requise",
    "install.status.successTitle": "Prêt",
    "install.status.idleMessage": "L'app open source installe; Hub livre seulement le bundle JSON.",
    "install.message.bundleDownloaded":
      "Bundle téléchargé. Configurez NEXT_PUBLIC_APP_SERVER_API_URL pour installer dans {target}.",
    "empty.title": "Aucun modèle disponible",
    "empty.description": "Démarrez Hub ou ajoutez des produits au catalogue.",
  },
  de: {
    "brand.subtitle": "Vorlagen-Hub",
    "nav.templates": "Vorlagen",
    "nav.docs": "Dokumente",
    "nav.connectHub": "Hub verbinden",
    "locale.label": "Sprache",
    "badge.officialTemplates": "Offizielle Vorlagen",
    "badge.bundleReady": "Bundle bereit",
    "hero.title": "Starte nützliche Business-Workspaces aus hochwertigen Vorlagen.",
    "hero.description":
      "Durchsuche kostenlose und bezahlte Vorlagen, prüfe den Funktionsbaum und installiere das Bundle in deinen Workspace.",
    "hero.explore": "Vorlagen ansehen",
    "hero.viewStructure": "Struktur anzeigen",
    "featured.label": "Empfohlene Vorlage",
    "featured.empty": "Noch keine Vorlagen",
    "catalog.title": "Vorlagenkatalog",
    "catalog.description": "Kuratierte Bundles für Einzelpersonen, Freelancer und kleine Teams.",
    "catalog.searchPlaceholder": "CRM, Retail, Aufgaben suchen...",
    "catalog.allCategories": "Alle Kategorien",
    "catalog.all": "Alle",
    "metric.functions": "Funktionen",
    "metric.docs": "Docs",
    "metric.official": "Offiziell",
    "common.yes": "Ja",
    "common.no": "Nein",
    "preview.title": "Bundle-Struktur",
    "preview.nodes": "{count} Knoten",
    "install.title": "Vorlage installieren",
    "install.description": "Von Hub herunterladen und in einen Workspace-Pfad installieren.",
    "install.template": "Vorlage",
    "install.downloadKey": "Download-Schlüssel",
    "install.downloadKeyPlaceholder": "Einmaligen Kaufschlüssel einfügen",
    "install.targetPath": "Zielpfad",
    "install.submit": "Abrufen und installieren",
    "install.loading": "Installiere...",
    "install.status.idleTitle": "Nutzt bestehenden Installer",
    "install.status.errorTitle": "Aktion erforderlich",
    "install.status.successTitle": "Bereit",
    "install.status.idleMessage": "Die Open-Source-App installiert; Hub liefert nur das Bundle JSON.",
    "install.message.bundleDownloaded":
      "Bundle heruntergeladen. NEXT_PUBLIC_APP_SERVER_API_URL konfigurieren, um in {target} zu installieren.",
    "empty.title": "Keine Vorlagen verfügbar",
    "empty.description": "Hub starten oder Produkte zum Katalog hinzufügen.",
  },
  ja: {
    "brand.subtitle": "テンプレート Hub",
    "nav.templates": "テンプレート",
    "nav.docs": "ドキュメント",
    "nav.connectHub": "Hub に接続",
    "locale.label": "言語",
    "badge.officialTemplates": "公式テンプレート",
    "badge.bundleReady": "Bundle 準備済み",
    "hero.title": "洗練されたテンプレートから業務ワークスペースをすばやく開始。",
    "hero.description":
      "無料・有料テンプレートを確認し、関数ツリーをプレビューして、能力 bundle をワークスペースへインストールします。",
    "hero.explore": "テンプレートを見る",
    "hero.viewStructure": "構造を見る",
    "featured.label": "注目テンプレート",
    "featured.empty": "テンプレートはまだありません",
    "catalog.title": "テンプレート一覧",
    "catalog.description": "個人、フリーランス、小規模チーム向けの厳選 bundle。",
    "catalog.searchPlaceholder": "CRM、小売、タスクを検索...",
    "catalog.allCategories": "すべてのカテゴリ",
    "catalog.all": "すべて",
    "metric.functions": "関数",
    "metric.docs": "文書",
    "metric.official": "公式",
    "common.yes": "はい",
    "common.no": "いいえ",
    "preview.title": "Bundle 構造",
    "preview.nodes": "{count} ノード",
    "install.title": "テンプレートをインストール",
    "install.description": "Hub からダウンロードしてワークスペースのパスへインストールします。",
    "install.template": "テンプレート",
    "install.downloadKey": "ダウンロードキー",
    "install.downloadKeyPlaceholder": "一回限りの購入キーを貼り付け",
    "install.targetPath": "インストール先パス",
    "install.submit": "取得してインストール",
    "install.loading": "インストール中...",
    "install.status.idleTitle": "既存インストーラーを使用",
    "install.status.errorTitle": "対応が必要",
    "install.status.successTitle": "準備完了",
    "install.status.idleMessage": "インストールはオープンソース側が担当し、Hub は bundle JSON のみ配布します。",
    "install.message.bundleDownloaded":
      "Bundle をダウンロードしました。{target} にインストールするには NEXT_PUBLIC_APP_SERVER_API_URL を設定してください。",
    "empty.title": "利用可能なテンプレートはありません",
    "empty.description": "Hub を起動するか catalog に製品を追加してください。",
  },
  ko: {
    "brand.subtitle": "템플릿 Hub",
    "nav.templates": "템플릿",
    "nav.docs": "문서",
    "nav.connectHub": "Hub 연결",
    "locale.label": "언어",
    "badge.officialTemplates": "공식 템플릿",
    "badge.bundleReady": "Bundle 준비됨",
    "hero.title": "완성도 높은 템플릿으로 업무 공간을 빠르게 시작하세요.",
    "hero.description":
      "무료 및 유료 템플릿을 살펴보고 함수 트리를 미리 본 뒤 capability bundle을 워크스페이스에 설치하세요.",
    "hero.explore": "템플릿 보기",
    "hero.viewStructure": "구조 보기",
    "featured.label": "추천 템플릿",
    "featured.empty": "템플릿 없음",
    "catalog.title": "템플릿 카탈로그",
    "catalog.description": "개인, 프리랜서, 소규모 팀을 위한 엄선된 bundle.",
    "catalog.searchPlaceholder": "CRM, 리테일, 작업 검색...",
    "catalog.allCategories": "전체 카테고리",
    "catalog.all": "전체",
    "metric.functions": "함수",
    "metric.docs": "문서",
    "metric.official": "공식",
    "common.yes": "예",
    "common.no": "아니요",
    "preview.title": "Bundle 구조",
    "preview.nodes": "{count}개 노드",
    "install.title": "템플릿 설치",
    "install.description": "Hub에서 다운로드해 워크스페이스 경로에 설치합니다.",
    "install.template": "템플릿",
    "install.downloadKey": "다운로드 키",
    "install.downloadKeyPlaceholder": "일회용 구매 키 붙여넣기",
    "install.targetPath": "대상 디렉터리 경로",
    "install.submit": "가져와 설치",
    "install.loading": "설치 중...",
    "install.status.idleTitle": "기존 설치기 사용",
    "install.status.errorTitle": "조치 필요",
    "install.status.successTitle": "준비됨",
    "install.status.idleMessage": "오픈소스 앱이 설치를 담당하고 Hub는 bundle JSON만 전달합니다.",
    "install.message.bundleDownloaded":
      "Bundle이 다운로드되었습니다. {target}에 설치하려면 NEXT_PUBLIC_APP_SERVER_API_URL을 설정하세요.",
    "empty.title": "사용 가능한 템플릿 없음",
    "empty.description": "Hub 서비스를 시작하거나 catalog에 제품을 추가하세요.",
  },
  "pt-BR": {
    "brand.subtitle": "Hub de modelos",
    "nav.templates": "Modelos",
    "nav.docs": "Docs",
    "nav.connectHub": "Conectar Hub",
    "locale.label": "Idioma",
    "badge.officialTemplates": "Modelos oficiais",
    "badge.bundleReady": "Bundle pronto",
    "hero.title": "Crie workspaces de negócio úteis a partir de modelos polidos.",
    "hero.description":
      "Explore modelos gratuitos e pagos, visualize a árvore de funções e instale o bundle no seu workspace.",
    "hero.explore": "Explorar modelos",
    "hero.viewStructure": "Ver estrutura",
    "featured.label": "Modelo em destaque",
    "featured.empty": "Ainda sem modelos",
    "catalog.title": "Catálogo de modelos",
    "catalog.description": "Bundles selecionados para indivíduos, freelancers e equipes pequenas.",
    "catalog.searchPlaceholder": "Buscar CRM, varejo, tarefas...",
    "catalog.allCategories": "Todas as categorias",
    "catalog.all": "Todos",
    "metric.functions": "Funções",
    "metric.docs": "Docs",
    "metric.official": "Oficial",
    "common.yes": "Sim",
    "common.no": "Não",
    "preview.title": "Estrutura do bundle",
    "preview.nodes": "{count} nós",
    "install.title": "Instalar modelo",
    "install.description": "Baixe pelo Hub e instale em um caminho do workspace.",
    "install.template": "Modelo",
    "install.downloadKey": "Chave de download",
    "install.downloadKeyPlaceholder": "Cole a chave de compra de uso único",
    "install.targetPath": "Caminho de destino",
    "install.submit": "Baixar e instalar",
    "install.loading": "Instalando...",
    "install.status.idleTitle": "Usa o instalador existente",
    "install.status.errorTitle": "Ação necessária",
    "install.status.successTitle": "Pronto",
    "install.status.idleMessage": "O app open source instala; o Hub só entrega o bundle JSON.",
    "install.message.bundleDownloaded":
      "Bundle baixado. Configure NEXT_PUBLIC_APP_SERVER_API_URL para instalar em {target}.",
    "empty.title": "Nenhum modelo disponível",
    "empty.description": "Inicie o Hub ou adicione produtos ao catálogo.",
  },
  ar: {
    "brand.subtitle": "مركز القوالب",
    "nav.templates": "القوالب",
    "nav.docs": "المستندات",
    "nav.connectHub": "ربط Hub",
    "locale.label": "اللغة",
    "badge.officialTemplates": "قوالب رسمية",
    "badge.bundleReady": "الحزمة جاهزة",
    "hero.title": "ابدأ مساحات عمل مفيدة من قوالب جاهزة عالية الجودة.",
    "hero.description":
      "تصفح القوالب المجانية والمدفوعة، وعاين شجرة الوظائف، ثم ثبّت حزمة القدرات في مساحة العمل.",
    "hero.explore": "استعراض القوالب",
    "hero.viewStructure": "عرض البنية",
    "featured.label": "قالب مميز",
    "featured.empty": "لا توجد قوالب بعد",
    "catalog.title": "كتالوج القوالب",
    "catalog.description": "حزم مختارة للأفراد والمستقلين والفرق الصغيرة.",
    "catalog.searchPlaceholder": "ابحث عن CRM أو التجزئة أو المهام...",
    "catalog.allCategories": "كل الفئات",
    "catalog.all": "الكل",
    "metric.functions": "الوظائف",
    "metric.docs": "المستندات",
    "metric.official": "رسمي",
    "common.yes": "نعم",
    "common.no": "لا",
    "preview.title": "بنية الحزمة",
    "preview.nodes": "{count} عقد",
    "install.title": "تثبيت القالب",
    "install.description": "نزّل من Hub وثبّت في مسار مساحة العمل.",
    "install.template": "القالب",
    "install.downloadKey": "مفتاح التنزيل",
    "install.downloadKeyPlaceholder": "الصق مفتاح الشراء لمرة واحدة",
    "install.targetPath": "مسار الدليل الهدف",
    "install.submit": "جلب وتثبيت",
    "install.loading": "جار التثبيت...",
    "install.status.idleTitle": "يستخدم المثبّت الحالي",
    "install.status.errorTitle": "يلزم إجراء",
    "install.status.successTitle": "جاهز",
    "install.status.idleMessage": "يتولى التطبيق مفتوح المصدر التثبيت، بينما يسلّم Hub ملف bundle JSON فقط.",
    "install.message.requestTicket": "جار طلب تذكرة تنزيل لمرة واحدة...",
    "install.message.downloadBundle": "جار تنزيل حزمة القدرات...",
    "install.message.installing": "جار التثبيت في مساحة العمل...",
    "install.message.installed": "تم تثبيت القالب بنجاح.",
    "install.message.bundleDownloaded":
      "تم تنزيل الحزمة. اضبط NEXT_PUBLIC_APP_SERVER_API_URL للتثبيت في {target}.",
    "install.error.rejectedKey": "رفض Hub مفتاح التنزيل ({status})",
    "install.error.noTicket": "لم يُرجع Hub تذكرة تنزيل",
    "install.error.downloadFailed": "فشل تنزيل الحزمة ({status})",
    "install.error.installFailed": "فشل طلب التثبيت ({status})",
    "install.error.generic": "فشل التثبيت",
    "empty.title": "لا توجد قوالب متاحة",
    "empty.description": "ابدأ خدمة Hub أو أضف منتجات إلى الكتالوج.",
  },
};

export function getLocaleConfig(locale: LocaleCode) {
  return locales.find((item) => item.code === locale) ?? locales[0];
}

export function isLocaleCode(value: string): value is LocaleCode {
  return locales.some((locale) => locale.code === value);
}

export function createTranslator(locale: LocaleCode) {
  const dictionary = dictionaries[locale] ?? {};
  return (key: TranslationKey, values?: Record<string, string | number>) => {
    let template = dictionary[key] ?? en[key] ?? key;
    if (values) {
      for (const [name, value] of Object.entries(values)) {
        template = template.replaceAll(`{${name}}`, String(value));
      }
    }
    return template;
  };
}
