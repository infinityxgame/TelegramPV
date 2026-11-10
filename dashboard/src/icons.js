import { h } from 'vue'
import { HugeiconsIcon } from '@hugeicons/vue'
import { FlashIcon } from '@hugeicons/core-free-icons'

const STROKE_WIDTH = 1.8

const createIcon = (glyph) => ({
  inheritAttrs: false,
  setup(_, { attrs }) {
    return () =>
      h(HugeiconsIcon, { icon: glyph, strokeWidth: STROKE_WIDTH, ...attrs })
  }
})

export const Zap = createIcon(FlashIcon)

export {
  Folder,
  Users,
  Radio,
  Bot,
  User,
  Search,
  Trash2,
  Download,
  RefreshCw,
  Settings,
  LogOut,
  KeyRound,
  Phone,
  ShieldCheck,
  CheckCircle2,
  AlertCircle,
  HelpCircle,
  ArrowRight,
  ArrowLeft,
  Loader2,
  Info,
  X,
  Menu,
  Power,
  Check,
  Layers,
  FileText,
  MessageSquare,
  ListFilter,
  CheckSquare,
  Square,
  Palette,
  Globe,
  Languages,
  ArrowUpRight,
  ExternalLink,
  ChevronRight,
  Settings2,
  ArrowDownToLine,
  Copy,
  Pause,
  Play,
  FileText as ScrollText
} from 'lucide-vue-next'
