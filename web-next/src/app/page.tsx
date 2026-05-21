import { HubMarketplace } from "@/components/hub/marketplace";
import { listHubProducts, type HubProduct } from "@/lib/hub-api";

const HUB_TAILWIND_CLASSES =
  "grid flex min-h-screen bg-[#f7f8fb] bg-[#f6f8fb] bg-[linear-gradient(180deg,#ffffff_0%,#f7fafc_100%)] border-b border-slate-100 border-emerald-200 bg-emerald-50 text-emerald-700 mx-auto h-16 max-w-7xl justify-between px-6 size-9 size-5 text-xs text-slate-500 hidden md:flex py-10 gap-10 lg:grid-cols-2 mb-5 flex-wrap max-w-3xl text-3xl text-4xl md:text-5xl font-semibold tracking-normal max-w-2xl text-base leading-7 mt-5 mt-7 sm:flex-row h-11 px-5 overflow-hidden items-start text-2xl pt-5 rounded bg-white py-0.5 size-3.5 text-amber-600 text-sky-600 py-8 lg:flex-row w-full sm:w-80 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400 pl-9 mb-3 text-lg min-h-12 grid-cols-3 border-y py-3 shrink-0 bg-slate-100 px-2.5 py-1.5 bg-slate-50 md:grid-cols-2 lg:sticky lg:top-6 lg:self-start space-y-4 read-only:cursor-default sr-only h-8 bg-transparent border-red-200 bg-red-50 text-red-800 border-emerald-200 bg-emerald-50 text-emerald-800 border-slate-200 bg-slate-50 text-slate-700 bg-white/85 bg-white/70 bg-white/95 bg-white/80 bg-white/90 bg-white/5 bg-white/10 border-white/10 text-white text-slate-100 text-slate-200 text-slate-300 text-slate-400 text-cyan-300 bg-rose-400 bg-amber-300 bg-emerald-400 backdrop-blur shadow-sm shadow-md shadow-lg shadow-slate-200/60 lg:grid-cols-[0.9fr_1.1fr] md:grid-cols-[1fr_220px] lg:grid-cols-[minmax(0,1fr)_420px] xl:grid-cols-2 divide-x line-clamp-2 uppercase tracking-wide hover:bg-white/10 hover:bg-slate-100 text-left px-3 py-2 border-emerald-300 ring-2 ring-emerald-100 leading-4";

export default async function Home() {
  void HUB_TAILWIND_CLASSES;

  let products: HubProduct[] = [];
  try {
    products = await listHubProducts();
  } catch {
    products = [];
  }

  return <HubMarketplace products={products} />;
}
