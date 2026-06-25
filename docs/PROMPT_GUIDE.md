# Image Generation Prompt Guide

How to write effective prompts for AI image generation with gimage.

## Core Principle

**Describe the scene, don't just list keywords.**

The model excels at language comprehension. Descriptive paragraphs consistently outperform disconnected word lists.

```bash
# Bad - keyword list
gimage generate "coffee mug, black, modern, studio"

# Good - narrative description
gimage generate "a minimalist ceramic coffee mug in matte black, resting on polished concrete, illuminated by soft three-point studio lighting"
```

## Prompt Strategies by Use Case

### Photorealistic Images

Use photography terminology:

| Element | Examples |
|---------|----------|
| Camera/Lens | "shot with 85mm lens", "wide-angle perspective" |
| Lighting | "golden hour light", "soft diffused lighting", "dramatic side lighting" |
| Mood | "serene atmosphere", "moody and cinematic" |
| Details | "fine texture visible", "shallow depth of field" |

```bash
gimage generate "portrait of an elderly craftsman in his woodworking shop, shot with 85mm lens, golden hour light streaming through dusty windows, shallow depth of field, warm tones"
```

### Stylized Illustrations

Be explicit about artistic style:

- **Art style**: kawaii, cel-shading, minimalist, watercolor, vector art
- **Line work**: bold outlines, thin linework, no outlines
- **Colors**: pastel palette, vibrant colors, monochromatic
- **Background**: transparent, gradient, solid color

```bash
gimage generate "kawaii style cat sticker with bold black outlines, pastel pink and white color palette, chibi proportions, transparent background"
```

### Text in Images

Gemini excels at legible text rendering:

- State the exact text to include in quotes
- Describe font style (not font name): "bold sans-serif", "elegant script"
- Specify the design context: logo, poster, menu, sign

```bash
gimage generate "vintage coffee shop menu board with chalk lettering reading 'Fresh Roasted Daily', weathered wooden frame, warm ambient lighting"
```

For professional text-heavy assets, Gemini 3 Pro is the default and best choice. Gemini 3.1 Flash offers the same native-resolution features at flash speed:
```bash
gimage generate "modern tech startup logo with text 'SKYWARD' in bold geometric sans-serif"
```

For native 4K resolution with sharp text:
```bash
# Gemini 3 Pro (premium quality)
gimage generate "detailed infographic about climate change" --image-size 4K

# Gemini 3.1 Flash (faster, tiered pricing)
gimage generate "detailed infographic about climate change" --model gemini-3.1-flash --image-size 4K
```

### Product Photography

Structure prompts with studio details:

1. **Product description**: materials, colors, size context
2. **Studio setup**: lighting type, background surface
3. **Camera angle**: overhead, eye-level, 45-degree
4. **Focus**: what should be sharp vs soft

```bash
gimage generate "luxury wristwatch with rose gold case and black leather strap, positioned at 45-degree angle on white marble surface, three-point softbox lighting, macro detail on watch face, clean white background"
```

## Gemini 3+ Advanced Features

These flags are exclusive to Gemini 3+ models (`gemini-3-pro-image`, `gemini-3.1-flash-image`). They are silently ignored on Gemini 2.5 Flash and non-Gemini providers, so it's safe to leave them in your default invocation.

### Compositional Editing with Reference Images

Pass one or more reference images via `--input-image` (repeatable) for Nano Banana-style compositional editing. The prompt then describes how to combine or transform them. Per-model caps: Gemini 2.5 Flash=3, Gemini 3 Pro=11, Gemini 3.1 Flash=14. PNG/JPEG/WebP only; each file is capped at 20 MB.

```bash
# Drop a product into a new scene
gimage generate "place this product on a marble counter, soft studio light" \
  --model gemini-3.1-flash --input-image product.png

# Combine a character and a background
gimage generate "this character standing in this environment, cinematic lighting" \
  --model gemini-3-pro --input-image character.png --input-image scene.jpg

# Multi-object composition (Gemini 3.1 Flash, up to 14 refs)
gimage generate "an office group photo of these people making funny faces" \
  --model gemini-3.1-flash \
  --input-image person1.png --input-image person2.png --input-image person3.png
```

**Tips:**
- The prompt should describe the *transformation*, not re-describe the references — the model already sees them.
- For "place X in Y" tasks, the FIRST reference image tends to anchor as the subject.
- Empty / whitespace-only paths are silently dropped, so it's safe to wire this through scripts that may not always have a reference.

### Thinking Mode

Gemini 3+ models can be told how much to "think" before generating. Higher levels improve layout, text rendering, and prompt adherence at the cost of latency.

```bash
# Low (default): fast, good for casual generation
gimage generate "abstract poster"

# Medium: better for layouts with multiple elements
gimage generate "menu board with 5 items and prices" --thinking medium

# High: best for text-heavy or complex composition
gimage generate "infographic: 'Q3 revenue up 18%' with 4 bullet points" \
  --model gemini-3-pro --image-size 4K --thinking high
```

**When to dial it up:**
- Text-heavy assets (logos, posters, infographics, menus)
- Complex multi-element compositions
- Geometric or precise layout (UI mockups, diagrams)

**When to leave it at default (or `minimal`):**
- Single-subject illustrations
- Stylistic exploration (you want creative variance, not careful planning)
- High-volume / cost-sensitive workloads

### Google Search Grounding

Enable `--grounding` to let Gemini pull real-time visual references from the web before generating. Billed per search query in addition to the per-image cost. Useful when the prompt references current entities (products, news, brands, places).

```bash
# Generate based on current real-world visual references
gimage generate "official Nintendo Switch 2 console front view, white background" \
  --model gemini-3-pro --grounding

# Anchor a stylized version to a real reference
gimage generate "watercolor painting of the Sydney Opera House at sunset" \
  --model gemini-3.1-flash --grounding
```

**When grounding helps:**
- Real products with specific brand visuals
- Recent / topical references (recently released games, current events)
- Public figures or landmarks where accuracy matters

**When to skip it:**
- Pure imagination / fantasy prompts (no real-world anchor needed)
- Prompts where the per-search billing isn't justified

## Quick Reference

### Transform Weak Prompts

| Weak | Strong |
|------|--------|
| "sunset" | "dramatic sunset over ocean waves, vibrant orange and purple sky, silhouetted palm trees, shot from beach level" |
| "cat" | "fluffy orange tabby cat curled up on vintage velvet armchair, soft window light, cozy living room atmosphere" |
| "logo" | "minimalist mountain logo in deep blue, clean geometric lines, text 'SUMMIT' in bold sans-serif below" |
| "food" | "artisan sourdough bread loaf on rustic wooden cutting board, steam rising, golden crust with flour dusting, warm kitchen background" |

### Style Modifiers

Add these to shift the aesthetic:

- **Realism**: "photorealistic", "hyperrealistic", "documentary style"
- **Art**: "oil painting style", "digital art", "pencil sketch"
- **Mood**: "cinematic", "ethereal", "gritty", "whimsical"
- **Era**: "1970s film grain", "art deco", "cyberpunk"
- **Quality**: "highly detailed", "4K resolution", "studio quality"

### Lighting Keywords

- **Soft**: diffused, overcast, ambient, fill light
- **Hard**: direct sunlight, spotlight, harsh shadows
- **Dramatic**: rim lighting, backlit, chiaroscuro
- **Natural**: golden hour, blue hour, midday sun
- **Studio**: softbox, ring light, three-point setup

## Size and Aspect Ratio

Match dimensions to content:

```bash
# Square (1024x1024) - social media, icons
gimage generate "..." --size 1024x1024

# Landscape (1792x1024) - banners, scenes
gimage generate "..." --size 1792x1024

# Portrait (1024x1792) - phone wallpapers, portraits
gimage generate "..." --size 1024x1792

# 4K with Gemini 3 Pro (premium)
gimage generate "..." --model gemini-3-pro --image-size 4K

# 4K with Gemini 3.1 Flash (faster, cheaper)
gimage generate "..." --model gemini-3.1-flash --image-size 4K
```

## Iteration Tips

1. **Start simple**, add details incrementally
2. **Be specific** about what matters most
3. **Use concrete nouns** over abstract concepts
4. **State what you don't want** directly in the prompt (e.g. "no text, no watermark")
5. **Try different phrasings** - small changes can yield different results

## Examples by Category

### Portraits
```bash
gimage generate "professional headshot of a confident business woman, navy blazer, neutral gray background, soft studio lighting, eye-level shot, warm skin tones"
```

### Landscapes
```bash
gimage generate "misty mountain valley at dawn, layers of blue ridges fading into distance, small river winding through pine forest, soft pink sky"
```

### Abstract
```bash
gimage generate "abstract fluid art with deep teal and copper metallic swirls, organic flowing shapes, high contrast, suitable for wall art"
```

### UI/Icons
```bash
gimage generate "flat design app icon for a meditation app, lotus flower symbol, gradient from soft purple to light blue, rounded square shape, minimal shadows"
```
