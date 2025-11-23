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

For professional text-heavy assets, use Gemini 3 Pro:
```bash
gimage generate "modern tech startup logo..." --model gemini-3-pro
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

# 4K with Gemini 3 Pro
gimage generate "..." --model gemini-3-pro --image-size 4K
```

## Iteration Tips

1. **Start simple**, add details incrementally
2. **Be specific** about what matters most
3. **Use concrete nouns** over abstract concepts
4. **Specify what you don't want** in the negative prompt (MCP only)
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
