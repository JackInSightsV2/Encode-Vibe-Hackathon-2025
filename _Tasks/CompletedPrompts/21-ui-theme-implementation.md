# UI Theme Implementation

## Overview
Implement a comprehensive theme system for the QT-1 middleware with five distinct themes: Light Mode, Dark Mode, Vibe Mode (70s Vice City), London Encode Mode, and Comet Opik Mode. Themes should be selectable from the login screen and maintain consistency across the entire web application.

## Priority: Medium
**Estimated Effort:** 3-4 days

## Technical Requirements
- [ ] CSS custom properties based theme system
- [ ] Theme selection from login screen
- [ ] Theme persistence in localStorage
- [ ] Smooth theme transitions
- [ ] Font stack implementation per theme
- [ ] Consistent theming across all components
- [ ] Theme-aware Tailwind configuration

## Implementation Checklist

### Theme Architecture
- [ ] Create theme context provider (`ThemeContext.tsx`)
- [ ] Implement theme hook (`useTheme.ts`)
- [ ] Set up CSS custom properties structure
- [ ] Create theme configuration object
- [ ] Add theme switching mechanism
- [ ] Implement theme persistence

### Theme Definitions

#### Light Mode (Default)
- [ ] Color Palette:
  - Primary: `#3B82F6` (Blue)
  - Secondary: `#64748B` (Slate)
  - Background: `#FFFFFF`
  - Surface: `#F8FAFC`
  - Text Primary: `#0F172A`
  - Text Secondary: `#475569`
- [ ] Font Stack:
  - Headings: Inter (600-700)
  - Body: Inter (400-500)
  - Monospace: Monaco, Menlo

#### Dark Mode
- [ ] Color Palette:
  - Primary: `#60A5FA` (Light Blue)
  - Secondary: `#94A3B8` (Light Slate)
  - Background: `#0F172A`
  - Surface: `#1E293B`
  - Text Primary: `#F1F5F9`
  - Text Secondary: `#CBD5E1`
- [ ] Font Stack: Same as Light Mode

#### Vibe Mode (70s Vice City)
- [ ] Color Palette:
  - Background: `#1E003B` (Deep indigo)
  - Sunset Pink: `#FF0080` (Electric magenta)
  - Teal Glow: `#00FFD1` (Neon aqua)
  - Sunset Gold: `#FF6F00` (Warm orange)
  - Surface: `#2E004F` (Dark purple)
- [ ] Font Stack:
  - Headings: Avant Garde Gothic or Neuropol
  - Body: Montserrat or Avenir Next
  - Display: Press Start 2P (micro-headings)
- [ ] Special Effects:
  - Gradient overlays (pink → purple, teal → blue)
  - Chrome/metallic button effects
  - Wireframe icons and neon borders

#### London Encode Mode
- [ ] Color Palette:
  - Background: `#0F172A` (Dark navy-blue)
  - Primary: `#3B82F6` (Bold Azure blue)
  - Secondary: `#94A3B8` (Cool gray)
  - Accent: `#06B6D4` (Electric cyan)
  - Surface: `#1E293B` (Dark panels)
- [ ] Font Stack:
  - Headings: JetBrains Mono or Space Mono
  - Body: Roboto Mono or Source Sans Pro
  - Accent: IBM Plex Mono (labels/badges)
- [ ] Special Effects:
  - Soft neon glows on hover
  - Neon-outline buttons
  - Edge-to-edge layouts

#### Comet Opik Mode
- [ ] Color Palette:
  - Primary: `#22C55E` (Vibrant teal from Opik)
  - Secondary: `#1F2937` (Dark charcoal)
  - Accent: `#F97316` (Warm orange)
  - Surface: `#FFFFFF` (Crisp white)
  - Muted BG: `#F3F4F6` (Light gray panels)
- [ ] Font Stack:
  - Headings: Poppins or Inter Bold
  - Body: Inter or Roboto
  - Monospace: Fira Code
- [ ] Special Effects:
  - Lots of white space
  - Subtle card shadows
  - Ghost buttons with hover lift

### Login Screen Theme Selector
- [ ] Create theme preview component
- [ ] Add theme selector to login UI
- [ ] Show live preview of each theme
- [ ] Implement smooth theme transition on selection
- [ ] Store theme preference before authentication

### Component Updates
- [ ] Update all base components for theme support:
  - [ ] Buttons (all variants)
  - [ ] Form inputs
  - [ ] Cards and panels
  - [ ] Navigation components
  - [ ] Modals and dialogs
  - [ ] Tables and lists
  - [ ] Charts and visualizations
- [ ] Update status indicators for theme compatibility
- [ ] Ensure proper contrast ratios in all themes
- [ ] Update icon colors per theme

### Theme-Specific Styling
- [ ] Create theme-specific CSS modules:
  - `themes/light.css`
  - `themes/dark.css`
  - `themes/vibe.css`
  - `themes/encode.css`
  - `themes/opik.css`
- [ ] Implement dynamic font loading
- [ ] Add theme-specific animations
- [ ] Create theme-aware gradients

### Tailwind Configuration
- [ ] Extend Tailwind config for theme support
- [ ] Create theme-aware utility classes
- [ ] Set up CSS variable integration
- [ ] Configure dynamic color palette

### Theme Switching UI
- [ ] Add theme switcher component
- [ ] Place in header/settings area
- [ ] Show current theme indicator
- [ ] Implement smooth transitions
- [ ] Add keyboard shortcuts for theme switching

### Special Theme Features

#### Vibe Mode Specifics
- [ ] Implement gradient dividers
- [ ] Add retro-futuristic UI elements
- [ ] Create neon glow effects
- [ ] Add scanline/CRT effects (optional)

#### Encode Mode Specifics
- [ ] Implement dark mode cards with neon glows
- [ ] Create minimalistic grid layouts
- [ ] Add tech-focused animations

#### Opik Mode Specifics
- [ ] Implement clean, spacious layouts
- [ ] Add subtle elevation effects
- [ ] Create developer-focused UI patterns

### Performance Considerations
- [ ] Lazy load theme-specific fonts
- [ ] Optimize CSS variable usage
- [ ] Minimize theme switching lag
- [ ] Preload critical theme assets
- [ ] Implement efficient theme caching

### Testing Requirements
- [ ] Test all components in all themes
- [ ] Verify color contrast accessibility
- [ ] Test theme persistence across sessions
- [ ] Validate theme switching performance
- [ ] Cross-browser theme compatibility
- [ ] Mobile responsive theme testing

## Acceptance Criteria
- [ ] All 5 themes are fully implemented
- [ ] Theme selection works from login screen
- [ ] Themes persist across sessions
- [ ] All components properly themed
- [ ] No visual inconsistencies
- [ ] Smooth theme transitions
- [ ] WCAG AA contrast compliance
- [ ] Font stacks properly loaded
- [ ] Theme-specific effects working

## Dependencies
- [ ] Existing authentication system (for login screen)
- [ ] Current component architecture
- [ ] Tailwind CSS setup

## Files to Modify/Create
- `frontend/src/contexts/ThemeContext.tsx` (new)
- `frontend/src/hooks/useTheme.ts` (new)
- `frontend/src/styles/themes/` (new directory)
- `frontend/src/components/auth/Login.tsx` (modify for theme selector)
- `frontend/src/components/ThemeSelector.tsx` (new)
- `frontend/src/App.tsx` (wrap with ThemeProvider)
- `frontend/tailwind.config.js` (extend for themes)
- `frontend/src/index.css` (add theme variables)
- All existing component files (theme support)