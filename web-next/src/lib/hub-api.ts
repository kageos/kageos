export type HubTreeNode = {
  relative_path: string;
  parent_path?: string;
  type: "package" | "function" | "docs" | string;
  code: string;
  name?: string;
  description?: string;
  tags?: string[];
  template_type?: string;
  method?: string;
  router?: string;
  sort_order?: number;
};

export type HubProduct = {
  id: string;
  slug: string;
  name: string;
  summary?: string;
  description?: string;
  category?: string;
  tags?: string[];
  currency?: string;
  price_cents?: number;
  version?: string;
  publisher?: string;
  preview?: {
    function_count?: number;
    doc_count?: number;
    tree_nodes?: HubTreeNode[];
  };
  created_at?: string;
  updated_at?: string;
};

type ProductsResponse = {
  products: HubProduct[];
};

export function getHubBaseUrl() {
  return process.env.NEXT_PUBLIC_HUB_API_URL || "http://127.0.0.1:8090";
}

export async function listHubProducts(): Promise<HubProduct[]> {
  const response = await fetch(`${getHubBaseUrl()}/api/v1/products`, {
    next: { revalidate: 30 },
  });
  if (!response.ok) {
    throw new Error(`Hub products request failed: ${response.status}`);
  }
  const data = (await response.json()) as ProductsResponse;
  return data.products ?? [];
}

export function formatPrice(product: HubProduct) {
  if (!product.price_cents || product.price_cents <= 0) {
    return "Free";
  }
  const amount = product.price_cents / 100;
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: product.currency || "USD",
    maximumFractionDigits: Number.isInteger(amount) ? 0 : 2,
  }).format(amount);
}
