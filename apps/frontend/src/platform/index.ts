import { createPlatformAdapters } from './adapters';
import { detectPlatform, readBrowserPlatformInput } from './detection';

export type * from './types';
export {
  UnsupportedPlatformCapabilityError,
  createPlatformAdapters,
  createWebPersistenceAdapter,
} from './adapters';
export { detectPlatform, readBrowserPlatformInput } from './detection';

export function getCurrentPlatformCapabilities() {
  return detectPlatform(readBrowserPlatformInput());
}

export const platformCapabilities = getCurrentPlatformCapabilities();
export const platformAdapters = createPlatformAdapters(platformCapabilities);
