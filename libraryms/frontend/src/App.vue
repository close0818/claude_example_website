<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import {
  ArrowDownToLine,
  ArrowLeftRight,
  ArrowUpRight,
  Bell,
  BookMarked,
  BookOpen,
  CalendarDays,
  ChartNoAxesCombined,
  Check,
  ChevronDown,
  CircleHelp,
  Clock3,
  LayoutDashboard,
  LibraryBig,
  Menu,
  Pencil,
  Plus,
  Search,
  Settings,
  Trash2,
  Users,
  X,
} from '@lucide/vue'

const navigation = [
  { id: 'overview', label: '總覽', icon: LayoutDashboard },
  { id: 'books', label: '館藏管理', icon: BookOpen },
  { id: 'members', label: '讀者名冊', icon: Users },
  { id: 'circulation', label: '借還作業', icon: ArrowLeftRight },
]

const activePage = ref('overview')
const mobileMenuOpen = ref(false)
const loading = ref(true)
const errorMessage = ref('')
const toastMessage = ref('')
const searchText = ref('')
const books = ref([])
const members = ref([])
const loans = ref([])
const stats = ref({ total_books: 0, available_books: 0, borrowed_books: 0, total_members: 0, active_loans: 0 })
const modalType = ref('')
const editingId = ref(null)
const busy = ref(false)
const bookForm = reactive({ isbn: '', title: '', author: '', category: '', status: 'available' })
const memberForm = reactive({ name: '', email: '', phone: '' })
const borrowForm = reactive({ book_id: '', member_id: '' })

const pageTitle = computed(() => navigation.find((item) => item.id === activePage.value)?.label ?? '總覽')
const availableBooks = computed(() => books.value.filter((book) => book.status === 'available'))
const activeLoans = computed(() => loans.value.filter((loan) => loan.status === 'borrowed'))
const filteredBooks = computed(() => {
  const query = searchText.value.trim().toLowerCase()
  if (!query) return books.value
  return books.value.filter((book) => [book.title, book.author, book.isbn, book.category].some((value) => value?.toLowerCase().includes(query)))
})
const filteredMembers = computed(() => {
  const query = searchText.value.trim().toLowerCase()
  if (!query) return members.value
  return members.value.filter((member) => [member.name, member.email, member.phone].some((value) => value?.toLowerCase().includes(query)))
})
const recentLoans = computed(() => loans.value.slice(0, 5))
const categorySummary = computed(() => {
  const totals = new Map()
  for (const book of books.value) totals.set(book.category, (totals.get(book.category) ?? 0) + 1)
  return [...totals.entries()].sort((a, b) => b[1] - a[1]).slice(0, 4)
})

async function request(path, options) {
  const response = await fetch(path, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...options?.headers },
  })
  const payload = await response.json().catch(() => ({}))
  if (!response.ok) throw new Error(payload.error || `請求失敗 (${response.status})`)
  return payload
}

async function loadData() {
  loading.value = true
  errorMessage.value = ''
  try {
    const [bookData, memberData, loanData, statsData] = await Promise.all([
      request('/api/books'),
      request('/api/members'),
      request('/api/borrow-records'),
      request('/api/stats'),
    ])
    books.value = bookData
    members.value = memberData
    loans.value = loanData
    stats.value = statsData
  } catch (error) {
    errorMessage.value = error.message || '無法連線到圖書館服務'
  } finally {
    loading.value = false
  }
}

function notify(message) {
  toastMessage.value = message
  window.setTimeout(() => {
    if (toastMessage.value === message) toastMessage.value = ''
  }, 2800)
}

function openBookModal(book = null) {
  editingId.value = book?.id ?? null
  Object.assign(bookForm, book ? {
    isbn: book.isbn,
    title: book.title,
    author: book.author,
    category: book.category,
    status: book.status,
  } : { isbn: '', title: '', author: '', category: '', status: 'available' })
  modalType.value = 'book'
}

function openMemberModal() {
  Object.assign(memberForm, { name: '', email: '', phone: '' })
  modalType.value = 'member'
}

async function saveBook() {
  busy.value = true
  try {
    await request(editingId.value ? `/api/books/${editingId.value}` : '/api/books', {
      method: editingId.value ? 'PUT' : 'POST',
      body: JSON.stringify(bookForm),
    })
    modalType.value = ''
    await loadData()
    notify(editingId.value ? '館藏資料已更新' : '新書已加入館藏')
  } catch (error) {
    notify(error.message)
  } finally {
    busy.value = false
  }
}

async function saveMember() {
  busy.value = true
  try {
    await request('/api/members', { method: 'POST', body: JSON.stringify(memberForm) })
    modalType.value = ''
    await loadData()
    notify('讀者資料已新增')
  } catch (error) {
    notify(error.message)
  } finally {
    busy.value = false
  }
}

async function deleteBook(book) {
  if (!window.confirm(`確定要刪除「${book.title}」嗎？`)) return
  try {
    await request(`/api/books/${book.id}`, { method: 'DELETE' })
    await loadData()
    notify('館藏資料已刪除')
  } catch (error) {
    notify(error.message)
  }
}

async function borrowBook() {
  if (!borrowForm.book_id || !borrowForm.member_id) {
    notify('請先選擇書籍與讀者')
    return
  }
  busy.value = true
  try {
    await request('/api/borrow', {
      method: 'POST',
      body: JSON.stringify({ book_id: Number(borrowForm.book_id), member_id: Number(borrowForm.member_id) }),
    })
    Object.assign(borrowForm, { book_id: '', member_id: '' })
    await loadData()
    notify('借閱登記完成，期限為 14 天')
  } catch (error) {
    notify(error.message)
  } finally {
    busy.value = false
  }
}

async function returnBook(loan) {
  busy.value = true
  try {
    await request(`/api/returns/${loan.id}`, { method: 'POST' })
    await loadData()
    notify(`「${loan.book_title}」已完成歸還`) 
  } catch (error) {
    notify(error.message)
  } finally {
    busy.value = false
  }
}

function formatDate(value) {
  if (!value) return '—'
  return new Intl.DateTimeFormat('zh-TW', { year: 'numeric', month: 'short', day: 'numeric' }).format(new Date(value))
}

function setPage(id) {
  activePage.value = id
  mobileMenuOpen.value = false
  searchText.value = ''
}

onMounted(loadData)
</script>

<template>
  <div class="app-shell">
    <aside class="sidebar" :class="{ 'sidebar-open': mobileMenuOpen }">
      <a class="brand" href="#overview" @click.prevent="setPage('overview')">
        <span class="brand-mark"><LibraryBig :size="21" :stroke-width="1.8" /></span>
        <span class="brand-copy"><strong>書庫</strong><small>LIBRARY DESK</small></span>
      </a>

      <div class="branch-label">工作空間</div>
      <button class="branch-switch" type="button">
        <span class="branch-avatar">中</span>
        <span class="branch-copy"><strong>中央圖書館</strong><small>管理員工作區</small></span>
        <ChevronDown :size="15" />
      </button>

      <div class="nav-label">館務管理</div>
      <nav class="main-nav" aria-label="主要導覽">
        <button
          v-for="item in navigation"
          :key="item.id"
          class="nav-item"
          :class="{ 'nav-item-active': activePage === item.id }"
          type="button"
          @click="setPage(item.id)"
        >
          <component :is="item.icon" :size="18" :stroke-width="1.8" />
          <span>{{ item.label }}</span>
          <span v-if="item.id === 'circulation' && stats.active_loans" class="nav-count">{{ stats.active_loans }}</span>
        </button>
      </nav>

      <div class="sidebar-bottom">
        <div class="sidebar-note">
          <span class="note-icon"><BookMarked :size="17" /></span>
          <div><strong>讓知識流動</strong><span>每一次借閱，都是新的開始。</span></div>
        </div>
        <button class="nav-item" type="button" @click="notify('設定功能即將推出')"><Settings :size="18" /><span>系統設定</span></button>
        <div class="profile-row">
          <div class="profile-avatar">林</div>
          <div class="profile-copy"><strong>林管理員</strong><span>館務管理者</span></div>
          <button class="icon-button profile-menu" aria-label="帳戶選單" @click="notify('帳戶選單')"><ChevronDown :size="16" /></button>
        </div>
      </div>
    </aside>

    <main class="workspace">
      <header class="topbar">
        <button class="icon-button mobile-menu" aria-label="開啟選單" @click="mobileMenuOpen = !mobileMenuOpen"><Menu :size="20" /></button>
        <div class="breadcrumb"><span>中央圖書館</span><span class="crumb-divider">/</span><strong>{{ pageTitle }}</strong></div>
        <div class="topbar-actions">
          <label v-if="activePage === 'books' || activePage === 'members'" class="top-search">
            <Search :size="17" />
            <input v-model="searchText" :placeholder="activePage === 'books' ? '搜尋書名、作者或 ISBN' : '搜尋讀者姓名或聯絡方式'" />
            <kbd>⌘ K</kbd>
          </label>
          <span class="today-label"><CalendarDays :size="16" />{{ formatDate(new Date()) }}</span>
          <button class="icon-button notification-button" aria-label="通知" @click="notify('目前沒有新通知')"><Bell :size="18" /><i></i></button>
          <button class="icon-button help-button" aria-label="說明" @click="notify('需要協助？請聯絡系統管理員')"><CircleHelp :size="18" /></button>
        </div>
      </header>

      <div class="page-content">
        <div v-if="errorMessage" class="connection-banner">
          <span class="connection-dot"></span>
          <span>無法連線至後端服務：{{ errorMessage }}</span>
          <button type="button" @click="loadData">重新連線</button>
        </div>

        <template v-if="activePage === 'overview'">
          <section class="welcome-row">
            <div><div class="eyebrow"><span class="eyebrow-line"></span>週三 · 館務概況</div><h1>早安，林管理員<span class="greeting-dot">。</span></h1><p>今天也讓好書找到下一位讀者。</p></div>
            <button class="button button-primary" type="button" @click="setPage('circulation')"><ArrowLeftRight :size="17" />登記借還</button>
          </section>

          <section class="stats-grid" aria-label="館藏統計">
            <article class="stat-card stat-card-books"><div class="stat-top"><span>館藏總數</span><span class="stat-icon"><BookOpen :size="18" /></span></div><div class="stat-value">{{ loading ? '—' : stats.total_books.toLocaleString() }}<small>冊</small></div><div class="stat-foot"><span class="stat-foot-dot"></span>目前收錄於館藏</div></article>
            <article class="stat-card stat-card-available"><div class="stat-top"><span>可借閱</span><span class="stat-icon"><Check :size="18" /></span></div><div class="stat-value">{{ loading ? '—' : stats.available_books.toLocaleString() }}<small>冊</small></div><div class="stat-foot"><span class="status-mark status-available"></span>準備好迎接新讀者</div></article>
            <article class="stat-card stat-card-loans"><div class="stat-top"><span>借閱中</span><span class="stat-icon"><ArrowLeftRight :size="18" /></span></div><div class="stat-value">{{ loading ? '—' : stats.active_loans.toLocaleString() }}<small>筆</small></div><div class="stat-foot"><span class="status-mark status-borrowed"></span>{{ stats.borrowed_books }} 冊離架中</div></article>
            <article class="stat-card stat-card-members"><div class="stat-top"><span>讀者總數</span><span class="stat-icon"><Users :size="18" /></span></div><div class="stat-value">{{ loading ? '—' : stats.total_members.toLocaleString() }}<small>位</small></div><div class="stat-foot"><span class="stat-foot-dot"></span>已登記讀者</div></article>
          </section>

          <section class="overview-grid">
            <article class="panel circulation-panel">
              <div class="panel-heading"><div><span class="section-kicker">CIRCULATION</span><h2>近期借閱</h2></div><button class="text-button" type="button" @click="setPage('circulation')">查看全部<ArrowUpRight :size="15" /></button></div>
              <div v-if="loading" class="empty-state">正在載入借閱資料…</div>
              <div v-else-if="!recentLoans.length" class="empty-state"><span class="empty-icon"><BookOpen :size="22" /></span><strong>尚無借閱紀錄</strong><span>讀者的第一筆借閱會出現在這裡</span></div>
              <div v-else class="loan-list">
                <div v-for="loan in recentLoans" :key="loan.id" class="loan-row">
                  <div class="book-cover-mini"><BookOpen :size="17" /></div>
                  <div class="loan-book"><strong>{{ loan.book_title }}</strong><span>{{ loan.member_name }} · {{ loan.isbn }}</span></div>
                  <div class="loan-due"><span :class="loan.status === 'borrowed' ? 'pill pill-active' : 'pill pill-returned'">{{ loan.status === 'borrowed' ? '借閱中' : '已歸還' }}</span><small>{{ loan.status === 'borrowed' ? `到期 ${formatDate(loan.due_at)}` : formatDate(loan.returned_at) }}</small></div>
                </div>
              </div>
            </article>

            <article class="panel collection-panel">
              <div class="panel-heading"><div><span class="section-kicker">COLLECTION</span><h2>館藏分類</h2></div><button class="icon-button panel-more" aria-label="館藏管理" @click="setPage('books')"><ArrowUpRight :size="17" /></button></div>
              <div v-if="categorySummary.length" class="category-list">
                <div v-for="([category, count], index) in categorySummary" :key="category" class="category-row">
                  <span class="category-dot" :class="`category-dot-${index + 1}`"></span><span class="category-name">{{ category }}</span><span class="category-count">{{ count }}<small> 冊</small></span>
                  <span class="category-track"><i :class="`category-fill category-fill-${index + 1}`" :style="{ width: `${Math.max(8, (count / Math.max(stats.total_books, 1)) * 100)}%` }"></i></span>
                </div>
              </div>
              <div v-else class="empty-state compact-empty">新增館藏後，分類概況會顯示在這裡。</div>
              <button class="collection-link" type="button" @click="setPage('books')"><span><ChartNoAxesCombined :size="16" />瀏覽完整館藏</span><ArrowUpRight :size="15" /></button>
            </article>
          </section>

          <section class="bottom-band">
            <div class="bottom-intro"><span class="section-kicker">QUICK ACCESS</span><h2>常用作業</h2><p>快速開始今天的館務流程</p></div>
            <button class="quick-action" type="button" @click="openBookModal()"><span class="quick-icon quick-icon-coral"><Plus :size="19" /></span><span><strong>新增館藏</strong><small>登記一本新書</small></span><ArrowUpRight :size="16" /></button>
            <button class="quick-action" type="button" @click="openMemberModal()"><span class="quick-icon quick-icon-green"><Users :size="18" /></span><span><strong>新增讀者</strong><small>建立讀者資料</small></span><ArrowUpRight :size="16" /></button>
            <button class="quick-action" type="button" @click="setPage('circulation')"><span class="quick-icon quick-icon-blue"><ArrowLeftRight :size="18" /></span><span><strong>借還登記</strong><small>處理書籍流通</small></span><ArrowUpRight :size="16" /></button>
          </section>
          <footer class="page-footer"><span>書庫 Library Desk</span><span>知識在此流動 <span class="footer-spark">✳</span></span></footer>
        </template>

        <template v-else-if="activePage === 'books'">
          <section class="page-heading-row"><div><div class="eyebrow"><span class="eyebrow-line"></span>COLLECTION</div><h1>館藏管理</h1><p>管理、搜尋與維護圖書館的每一本書。</p></div><button class="button button-primary" type="button" @click="openBookModal()"><Plus :size="17" />新增館藏</button></section>
          <section class="table-panel">
            <div class="table-toolbar"><div><strong>全部館藏</strong><span class="record-count">{{ filteredBooks.length }} 本書</span></div><button class="button button-quiet" type="button" @click="loadData"><ArrowDownToLine :size="16" />重新整理</button></div>
            <div class="table-scroll"><table><thead><tr><th>書籍</th><th>ISBN</th><th>分類</th><th>狀態</th><th class="actions-col">操作</th></tr></thead><tbody>
              <tr v-for="book in filteredBooks" :key="book.id"><td><div class="table-book"><span class="table-book-icon"><BookOpen :size="17" /></span><span><strong>{{ book.title }}</strong><small>{{ book.author }}</small></span></div></td><td class="muted-cell">{{ book.isbn }}</td><td><span class="category-tag">{{ book.category }}</span></td><td><span class="pill" :class="book.status === 'available' ? 'pill-active' : 'pill-borrowed'"><i></i>{{ book.status === 'available' ? '可借閱' : '借閱中' }}</span></td><td><div class="row-actions"><button class="icon-button" :aria-label="`編輯${book.title}`" @click="openBookModal(book)"><Pencil :size="16" /></button><button class="icon-button danger-action" :aria-label="`刪除${book.title}`" @click="deleteBook(book)"><Trash2 :size="16" /></button></div></td></tr>
              <tr v-if="!loading && !filteredBooks.length"><td colspan="5"><div class="table-empty"><BookOpen :size="24" /><strong>{{ searchText ? '找不到符合的館藏' : '還沒有館藏資料' }}</strong><span>{{ searchText ? '試試其他書名、作者或 ISBN' : '新增第一本書，開始建立館藏' }}</span></div></td></tr>
            </tbody></table></div>
            <div class="table-foot"><span>顯示 {{ filteredBooks.length }} 筆館藏</span><span>資料即時同步</span></div>
          </section>
        </template>

        <template v-else-if="activePage === 'members'">
          <section class="page-heading-row"><div><div class="eyebrow"><span class="eyebrow-line"></span>MEMBERS</div><h1>讀者名冊</h1><p>查看已登記讀者與聯絡資料。</p></div><button class="button button-primary" type="button" @click="openMemberModal()"><Plus :size="17" />新增讀者</button></section>
          <section class="table-panel">
            <div class="table-toolbar"><div><strong>已登記讀者</strong><span class="record-count">{{ filteredMembers.length }} 位讀者</span></div><button class="button button-quiet" type="button" @click="loadData"><ArrowDownToLine :size="16" />重新整理</button></div>
            <div class="table-scroll"><table><thead><tr><th>讀者</th><th>電子郵件</th><th>電話</th><th>加入日期</th><th>借閱狀態</th></tr></thead><tbody>
              <tr v-for="(member, index) in filteredMembers" :key="member.id"><td><div class="member-cell"><span class="member-avatar" :class="`member-avatar-${index % 4 + 1}`">{{ member.name.slice(0, 1) }}</span><strong>{{ member.name }}</strong></div></td><td class="muted-cell">{{ member.email }}</td><td class="muted-cell">{{ member.phone }}</td><td class="muted-cell">{{ formatDate(member.created_at) }}</td><td><span class="member-loan-count"><BookMarked :size="15" />{{ loans.filter((loan) => loan.member_id === member.id && loan.status === 'borrowed').length }} 本借閱中</span></td></tr>
              <tr v-if="!loading && !filteredMembers.length"><td colspan="5"><div class="table-empty"><Users :size="24" /><strong>{{ searchText ? '找不到符合的讀者' : '還沒有讀者資料' }}</strong><span>新增讀者以開始辦理借閱</span></div></td></tr>
            </tbody></table></div>
            <div class="table-foot"><span>顯示 {{ filteredMembers.length }} 位讀者</span><span>資料即時同步</span></div>
          </section>
        </template>

        <template v-else>
          <section class="page-heading-row"><div><div class="eyebrow"><span class="eyebrow-line"></span>CIRCULATION</div><h1>借還作業</h1><p>辦理圖書借出與歸還，借期為 14 天。</p></div><div class="active-loan-total"><Clock3 :size="17" /><span>目前借閱中</span><strong>{{ activeLoans.length }}</strong></div></section>
          <section class="circulation-layout">
            <article class="panel borrow-panel"><div class="panel-heading"><div><span class="section-kicker">NEW CHECKOUT</span><h2>登記借閱</h2></div><span class="borrow-icon"><ArrowLeftRight :size="19" /></span></div><p class="form-intro">選擇可借閱書籍與讀者，系統將自動設定 14 天借期。</p>
              <form class="stack-form" @submit.prevent="borrowBook"><label>選擇書籍<select v-model="borrowForm.book_id" required><option value="" disabled>選擇一本可借閱的書</option><option v-for="book in availableBooks" :key="book.id" :value="book.id">{{ book.title }} · {{ book.author }}</option></select></label><label>借閱讀者<select v-model="borrowForm.member_id" required><option value="" disabled>選擇讀者</option><option v-for="member in members" :key="member.id" :value="member.id">{{ member.name }} · {{ member.email }}</option></select></label><div class="due-note"><CalendarDays :size="16" /><span>預計歸還日</span><strong>{{ formatDate(new Date(Date.now() + 14 * 86400000)) }}</strong></div><button class="button button-primary form-submit" type="submit" :disabled="busy || !availableBooks.length || !members.length"><ArrowLeftRight :size="17" />確認借閱</button><p v-if="!availableBooks.length" class="form-hint">目前沒有可借閱的書籍</p><p v-else-if="!members.length" class="form-hint">請先新增讀者資料</p></form>
            </article>
            <article class="panel active-panel"><div class="panel-heading"><div><span class="section-kicker">ON LOAN</span><h2>流通中的書籍</h2></div><span class="loan-counter">{{ activeLoans.length }} 筆</span></div>
              <div v-if="!activeLoans.length" class="empty-state circulation-empty"><span class="empty-icon"><Check :size="22" /></span><strong>目前沒有未歸還的書</strong><span>新的借閱會顯示在這裡</span></div>
              <div v-else class="active-loan-list"><div v-for="loan in activeLoans" :key="loan.id" class="active-loan-row"><span class="loan-book-icon"><BookOpen :size="18" /></span><div class="active-loan-info"><strong>{{ loan.book_title }}</strong><span>{{ loan.member_name }} · ISBN {{ loan.isbn }}</span><small><Clock3 :size="13" />應歸還於 {{ formatDate(loan.due_at) }}</small></div><button class="button button-return" type="button" :disabled="busy" @click="returnBook(loan)"><Check :size="15" />辦理歸還</button></div></div>
            </article>
          </section>
          <section class="panel history-panel"><div class="panel-heading"><div><span class="section-kicker">HISTORY</span><h2>借閱紀錄</h2></div><span class="record-count">共 {{ loans.length }} 筆</span></div><div class="table-scroll"><table><thead><tr><th>書籍</th><th>讀者</th><th>借出日期</th><th>應還日期</th><th>狀態</th></tr></thead><tbody><tr v-for="loan in loans" :key="loan.id"><td><div class="table-book"><span class="table-book-icon"><BookOpen :size="17" /></span><span><strong>{{ loan.book_title }}</strong><small>{{ loan.isbn }}</small></span></div></td><td class="muted-cell">{{ loan.member_name }}</td><td class="muted-cell">{{ formatDate(loan.borrowed_at) }}</td><td class="muted-cell">{{ formatDate(loan.due_at) }}</td><td><span class="pill" :class="loan.status === 'borrowed' ? 'pill-borrowed' : 'pill-active'"><i></i>{{ loan.status === 'borrowed' ? '借閱中' : '已歸還' }}</span></td></tr><tr v-if="!loans.length"><td colspan="5"><div class="table-empty"><ArrowLeftRight :size="24" /><strong>尚無借閱紀錄</strong><span>完成第一筆借閱後會顯示在這裡</span></div></td></tr></tbody></table></div></section>
        </template>
      </div>
    </main>

    <div v-if="mobileMenuOpen" class="sidebar-scrim" @click="mobileMenuOpen = false"></div>
    <div v-if="modalType" class="modal-scrim" @click.self="modalType = ''" @keydown.esc="modalType = ''">
      <section class="modal" role="dialog" aria-modal="true" :aria-labelledby="'modal-title'">
        <div class="modal-heading"><div><span class="section-kicker">{{ modalType === 'book' ? 'COLLECTION' : 'MEMBERS' }}</span><h2 id="modal-title">{{ modalType === 'book' ? (editingId ? '編輯館藏' : '新增館藏') : '新增讀者' }}</h2></div><button class="icon-button" aria-label="關閉" @click="modalType = ''"><X :size="19" /></button></div>
        <form v-if="modalType === 'book'" class="stack-form modal-form" @submit.prevent="saveBook"><label>書名<input v-model="bookForm.title" required placeholder="輸入書名" /></label><div class="form-two-col"><label>作者<input v-model="bookForm.author" required placeholder="作者姓名" /></label><label>ISBN<input v-model="bookForm.isbn" required placeholder="ISBN 編號" /></label></div><label>分類<input v-model="bookForm.category" required placeholder="例如：文學、歷史、科學" /></label><label v-if="editingId">館藏狀態<select v-model="bookForm.status"><option value="available">可借閱</option><option value="borrowed">借閱中</option></select></label><div class="modal-actions"><button class="button button-quiet" type="button" @click="modalType = ''">取消</button><button class="button button-primary" type="submit" :disabled="busy"><Plus v-if="!editingId" :size="16" /><Check v-else :size="16" />{{ editingId ? '儲存變更' : '新增館藏' }}</button></div></form>
        <form v-else class="stack-form modal-form" @submit.prevent="saveMember"><label>姓名<input v-model="memberForm.name" required placeholder="讀者姓名" /></label><label>電子郵件<input v-model="memberForm.email" type="email" required placeholder="name@example.com" /></label><label>聯絡電話<input v-model="memberForm.phone" type="tel" required placeholder="電話號碼" /></label><div class="modal-actions"><button class="button button-quiet" type="button" @click="modalType = ''">取消</button><button class="button button-primary" type="submit" :disabled="busy"><Plus :size="16" />新增讀者</button></div></form>
      </section>
    </div>
    <Transition name="toast"><div v-if="toastMessage" class="toast-message"><Check :size="17" />{{ toastMessage }}</div></Transition>
  </div>
</template>
