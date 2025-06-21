import { FontLoadingState } from '../types/theme';

export class FontLoader {
  private static instance: FontLoader;
  private loadingState: FontLoadingState = {};
  private loadedFonts = new Set<string>();

  static getInstance(): FontLoader {
    if (!FontLoader.instance) {
      FontLoader.instance = new FontLoader();
    }
    return FontLoader.instance;
  }

  async loadFont(fontFamily: string, fontUrl?: string): Promise<boolean> {
    if (this.loadedFonts.has(fontFamily)) {
      return true;
    }

    if (this.loadingState[fontFamily]?.loading) {
      return new Promise((resolve) => {
        const checkLoaded = () => {
          if (this.loadingState[fontFamily]?.loaded) {
            resolve(true);
          } else if (this.loadingState[fontFamily]?.error) {
            resolve(false);
          } else {
            setTimeout(checkLoaded, 100);
          }
        };
        checkLoaded();
      });
    }

    this.loadingState[fontFamily] = { loaded: false, loading: true };

    try {
      // Check if font is already available
      if (await this.checkFontAvailable(fontFamily)) {
        this.loadingState[fontFamily] = { loaded: true, loading: false };
        this.loadedFonts.add(fontFamily);
        return true;
      }

      // If URL provided, load from URL
      if (fontUrl) {
        await this.loadFontFromUrl(fontFamily, fontUrl);
      } else {
        // Try loading from Google Fonts
        await this.loadFromGoogleFonts(fontFamily);
      }

      this.loadingState[fontFamily] = { loaded: true, loading: false };
      this.loadedFonts.add(fontFamily);
      return true;
    } catch (error) {
      this.loadingState[fontFamily] = { 
        loaded: false, 
        loading: false, 
        error: error instanceof Error ? error.message : 'Unknown error' 
      };
      return false;
    }
  }

  private async checkFontAvailable(fontFamily: string): Promise<boolean> {
    if (!document.fonts) return false;
    
    try {
      await document.fonts.load(`12px "${fontFamily}"`);
      const face = new FontFace(fontFamily, 'local("' + fontFamily + '")');
      await face.load();
      return face.status === 'loaded';
    } catch {
      return false;
    }
  }

  private async loadFontFromUrl(fontFamily: string, fontUrl: string): Promise<void> {
    const fontFace = new FontFace(fontFamily, `url(${fontUrl})`);
    await fontFace.load();
    document.fonts.add(fontFace);
  }

  private async loadFromGoogleFonts(fontFamily: string): Promise<void> {
    const existingLink = document.querySelector(`link[data-font="${fontFamily}"]`);
    if (existingLink) return;

    const weights = ['300', '400', '500', '600', '700'];
    const googleFontUrl = `https://fonts.googleapis.com/css2?family=${fontFamily.replace(' ', '+')}:wght@${weights.join(';')}&display=swap`;

    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = googleFontUrl;
    link.setAttribute('data-font', fontFamily);
    
    return new Promise((resolve, reject) => {
      link.onload = () => resolve();
      link.onerror = () => reject(new Error(`Failed to load font: ${fontFamily}`));
      document.head.appendChild(link);
    });
  }

  getFontLoadingState(fontFamily: string) {
    return this.loadingState[fontFamily] || { loaded: false, loading: false };
  }

  isLoaded(fontFamily: string): boolean {
    return this.loadedFonts.has(fontFamily);
  }

  async loadMultipleFonts(fonts: string[]): Promise<boolean[]> {
    return Promise.all(fonts.map(font => this.loadFont(font)));
  }
}

export const fontLoader = FontLoader.getInstance();

export const preloadThemeFonts = async (fonts: { heading: string; body: string; mono: string; display?: string }): Promise<void> => {
  const fontsToLoad = [fonts.heading, fonts.body, fonts.mono];
  if (fonts.display) fontsToLoad.push(fonts.display);

  await fontLoader.loadMultipleFonts(fontsToLoad);
};