import { buildLLMSIndex } from '@/lib/source';

export const revalidate = false;

export function GET() {
  return new Response(buildLLMSIndex());
}
