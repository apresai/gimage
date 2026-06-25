# Gimage MCP Real-World Examples

Practical examples demonstrating how to use gimage MCP tools with Claude and other AI assistants.

## Table of Contents

1. [Quick Start Examples](#quick-start-examples)
2. [Image Generation Workflows](#image-generation-workflows)
3. [Image Processing Pipelines](#image-processing-pipelines)
4. [Batch Processing Scenarios](#batch-processing-scenarios)
5. [Web Development Use Cases](#web-development-use-cases)
6. [E-commerce Workflows](#e-commerce-workflows)
7. [Social Media Content](#social-media-content)
8. [Creative Content Generation](#creative-content-generation)
9. [Advanced Multi-Step Workflows](#advanced-multi-step-workflows)

---

## Quick Start Examples

### Example 1: Generate Your First AI Image

**User Prompt:**
```
Generate an image of a sunset over mountain peaks with orange and purple sky
```

**What Happens:**
1. Claude calls `generate_image` tool with your prompt
2. Uses Gemini 3 Pro (default, high quality with sharp text)
3. Generates 1024x1024 image
4. Saves to current directory with timestamp

**Result:**
```
Generated image saved to: generated_1730400123.png
Size: 1024x1024
Model: gemini-3-pro-image
```

**Want lower-cost generation?** Specify `model="gemini-2.5-flash-image"` for the lowest-cost Google model.

### Example 2: Resize an Existing Photo

**User Prompt:**
```
I have vacation-photo.jpg that's 4000x3000. Please resize it to 1920x1080 for my website.
```

**What Happens:**
1. Claude calls `resize_image` tool
2. Resizes using high-quality Lanczos resampling
3. Saves as vacation-photo_resized.jpg

**Result:**
```
✓ Resized image saved to: vacation-photo_resized.jpg
Original size: 4000x3000
New size: 1920x1080
```

### Example 3: Compress Multiple Images

**User Prompt:**
```
I have a folder called 'blog-photos' with 50 images. Compress them all to 85% quality to save space.
```

**What Happens:**
1. Claude calls `batch_compress` tool
2. Processes all images concurrently
3. Saves to 'blog-photos-compressed' directory
4. Reports total space saved

**Result:**
```
✓ Batch compress complete
Processed: 50 images
Failed: 0
Total original size: 156.8 MB
Total compressed size: 28.4 MB
Savings: 128.4 MB (81.9%)
```

---

## Image Generation Workflows

### Example 4: Generate Multiple Style Variations

**User Prompt:**
```
Generate 3 different versions of a fantasy forest scene:
1. Photorealistic style with detailed trees and lighting
2. Artistic watercolor painting style
3. Anime style with vibrant colors

Use 1024x1024 size for all.
```

**Claude's Actions:**
```
I'll generate three different style variations of a fantasy forest scene for you.

[Calls generate_image three times with different style parameters]
```

**Results:**
- `forest_photorealistic.png` - Highly detailed, natural lighting
- `forest_watercolor.png` - Soft edges, artistic interpretation
- `forest_anime.png` - Bold colors, stylized rendering

**Follow-up Prompt:**
```
I like the anime version best. Can you generate 3 more variations with different seasons - spring, summer, winter?
```

### Example 5: Reproducible Generation with Seeds

**User Prompt:**
```
Generate an image of a dragon with seed 12345. Then generate the exact same image again to verify reproducibility.
```

**What This Demonstrates:**
- Using seeds for consistent results
- Useful for A/B testing or generating variations
- Same seed + same prompt = identical image

**Claude's Response:**
```
I'll generate the dragon image twice with the same seed to demonstrate reproducibility.

[Calls generate_image with seed=12345 twice]

Both images are identical. Seeds ensure reproducible results, which is useful when you want to:
- Regenerate an image you liked
- Share generation parameters with others
- Create consistent variations
```

### Example 6: Stating Exclusions in the Prompt

**User Prompt:**
```
Generate a beautiful garden scene, but I don't want any people, buildings, or modern objects in it. Just nature.
```

**Claude's Actions:**
```
I'll generate a garden scene with the exclusions stated directly in the prompt.

[Calls generate_image with prompt="A beautiful garden scene with no people, no buildings, no modern objects, no cars, no technology. Pure nature only."]
```

**Result:**
Pure nature scene without any human elements or modern objects.

### Example 7: High-Resolution Generation with Gemini 3 Pro

**User Prompt:**
```
I need a professional 2K image for print. Generate a hyper-realistic mountain landscape using the highest quality model available.
```

**Claude's Actions:**
```
For professional print quality, I'll use Gemini 3 Pro with native 4K output.

[Calls generate_image with model="gemini-3-pro-image", image_size="4K"]

Note: This uses Gemini API authentication and the 4K Gemini 3 Pro pricing tier.
```

---

## Gemini 3+ Advanced Features (Compositional Editing, Thinking, Grounding)

These three capabilities are exclusive to Gemini 3+ models (`gemini-3-pro-image`, `gemini-3.1-flash-image`). They are silently ignored on other providers, so passing them is always safe.

### Example 7a: Compositional Editing with One Reference Image

**User Prompt:**
```
I have a product photo at ~/Pictures/sneaker.png. Place it on a marble counter under soft studio lighting and save the result to ~/Desktop/sneaker-styled.png.
```

**Claude's Actions:**
```
Using Gemini 3.1 Flash for Nano Banana-style compositional editing.

[Calls generate_image with prompt="place this product on a marble counter, soft three-point studio light, shallow depth of field", model="gemini-3.1-flash-image", input_images=["~/Pictures/sneaker.png"], image_size="2K", output="~/Desktop/sneaker-styled.png"]

The original product is preserved in the composition; only the environment and lighting are generated.
```

**Result:**
The original sneaker stays visually consistent and appears on a freshly-generated marble background with the requested lighting.

---

### Example 7b: Multi-Reference Composition

**User Prompt:**
```
Combine these three reference images — a character (character.png), a background scene (forest.jpg), and a prop (sword.png) — into a single cinematic shot of the character in the forest holding the sword.
```

**Claude's Actions:**
```
Using Gemini 3.1 Flash with three reference images for multi-object composition. The model can take up to 14 references on this variant.

[Calls generate_image with prompt="this character standing in this forest holding this sword, cinematic lighting, low angle, dramatic shadows", model="gemini-3.1-flash-image", input_images=["character.png", "forest.jpg", "sword.png"], aspect_ratio="16:9", output="~/Desktop/composed.png"]

All three references contribute to the composition. The prompt describes how to combine them, not what they look like.
```

**Per-model caps**: Gemini 2.5 Flash=3, Gemini 3 Pro=11, Gemini 3.1 Flash=14. Exceeding the cap returns a clear error before any HTTP call.

---

### Example 7c: Thinking Mode for Complex Layouts

**User Prompt:**
```
Generate a Q3 2026 revenue infographic with these data points: revenue up 18%, gross margin up 4pp, 3 new product launches. Use a clean corporate style with the company colors navy and gold.
```

**Claude's Actions:**
```
This is a text-heavy multi-element layout — exactly where thinking mode pays off. Using Gemini 3 Pro at 4K with high thinking level for careful text rendering and composition planning.

[Calls generate_image with prompt="corporate infographic for Q3 2026: 'Revenue up 18%' as headline, three labeled sections — 'Gross Margin +4pp', '3 New Product Launches', and a small trend chart. Navy and gold palette, modern sans-serif typography, clean white background", model="gemini-3-pro-image", image_size="4K", thinking="high", output="~/Desktop/q3-infographic.png"]

Higher thinking levels add latency but materially improve text legibility and element alignment on dense layouts.
```

**When to use each level**:
- `minimal`: high-volume cost-sensitive workloads, single-subject illustrations
- `low` (default): casual generation, stylistic exploration
- `medium`: layouts with multiple labeled elements
- `high`: text-heavy assets, precise compositions, UI mockups

---

### Example 7d: Google Search Grounding for Current References

**User Prompt:**
```
Generate an official-looking product photo of the Nintendo Switch 2 console, front view, white background. Make sure the design is accurate to the current release.
```

**Claude's Actions:**
```
The user wants an accurate rendering of a real product. Enabling Google Search grounding so Gemini can pull current visual references before generating. Note: each search query inside grounding is billed in addition to the image cost.

[Calls generate_image with prompt="Nintendo Switch 2 console, front view, white background, product photography style", model="gemini-3-pro-image", grounding=true, output="~/Desktop/switch2.png"]

Grounding is most useful for products, public figures, and landmarks where visual accuracy matters. For pure imagination prompts, skip it to avoid the extra search billing.
```

**Result:**
The generated image matches the official Switch 2 design rather than hallucinating an imagined version.

---

## Image Processing Pipelines

### Example 8: Prepare Photo for Instagram

**User Prompt:**
```
I have portrait.jpg (3000x4000). Prepare it for Instagram:
- Crop to square 1080x1080 from the center
- Compress to 90% quality
- Save as instagram-ready.jpg
```

**Claude's Multi-Step Process:**
```
I'll prepare your photo for Instagram in three steps.

Step 1: Calculate center crop coordinates
[Image is 3000x4000, so crop from (960, 1460) with 1080x1080 size]

Step 2: Crop to square
[Calls crop_image with x=960, y=1460, width=1080, height=1080]

Step 3: Compress
[Calls compress_image with quality=90]

✓ Instagram-ready image saved to: instagram-ready.jpg
Size: 1080x1080
Quality: 90%
File size: 245 KB
```

### Example 9: Optimize Images for Web Performance

**User Prompt:**
```
I have 30 product images in 'products-raw' folder (all PNG, 5000x5000 each). Optimize them for web:
1. Resize to maximum 1200x1200
2. Convert to WebP format
3. Save in 'products-optimized' folder
```

**Claude's Workflow:**
```
I'll optimize your product images for web in two batch operations.

Step 1: Batch resize all images to 1200x1200
[Calls batch_resize with width=1200, height=1200]
✓ Resized 30 images to 1200x1200

Step 2: Batch convert to WebP format
[Calls batch_convert with format="webp"]
✓ Converted 30 images to WebP

Results:
- Original: 30 PNG files, 450 MB total
- Optimized: 30 WebP files, 38 MB total
- Savings: 412 MB (91.6% reduction)
- Load time improvement: ~12x faster
```

### Example 10: Create Responsive Image Set

**User Prompt:**
```
Take hero-image.jpg and create a responsive image set for my website:
- Large: 1920x1080 (desktop)
- Medium: 1024x576 (tablet)
- Small: 640x360 (mobile)
- Thumbnail: 320x180 (preview)

Compress all to 85% quality and convert to WebP.
```

**Claude's Detailed Process:**
```
I'll create a complete responsive image set with 4 sizes.

[Calls resize_image 4 times with different dimensions]
[Calls compress_image 4 times with quality=85]
[Calls convert_image 4 times to WebP format]

✓ Responsive image set created:
- hero-large.webp (1920x1080, 156 KB)
- hero-medium.webp (1024x576, 45 KB)
- hero-small.webp (640x360, 18 KB)
- hero-thumb.webp (320x180, 6 KB)

You can now use these in responsive HTML with srcset:
<img srcset="hero-small.webp 640w, hero-medium.webp 1024w, hero-large.webp 1920w"
     src="hero-medium.webp" alt="Hero image">
```

---

## Batch Processing Scenarios

### Example 11: Process Photography Portfolio

**User Prompt:**
```
I'm a photographer with 200 RAW photos exported as TIFF files in 'portfolio-raw'. I need to:
1. Convert all to JPG format
2. Resize to 2560x1440 (QHD resolution)
3. Compress to 95% quality (maintain quality)
4. Use 8 parallel workers for speed

Save results in 'portfolio-web'.
```

**Claude's Execution:**
```
I'll process your portfolio in an optimized batch workflow.

[Calls batch_convert with format="jpg", workers=8]
✓ Converted 200 TIFF → JPG

[Calls batch_resize with width=2560, height=1440, workers=8]
✓ Resized 200 images to 2560x1440

[Calls batch_compress with quality=95, workers=8]
✓ Compressed 200 images to 95% quality

Final Results:
- Processed: 200 images
- Time: ~3 minutes (8 workers)
- Original size: 4.2 GB (TIFF)
- Final size: 280 MB (JPG)
- Savings: 3.92 GB (93.3%)
- Average file: 1.4 MB per image
```

### Example 12: Prepare Images for CDN Deployment

**User Prompt:**
```
I have 500 blog images in various formats (PNG, JPG, BMP) in 'blog-images'. Standardize them for CDN:
- All to WebP format
- Maximum 1600x1200 resolution
- 85% quality
- Output to 'cdn-ready'
```

**Claude's Strategy:**
```
I'll standardize your blog images for optimal CDN delivery.

Analysis: Found 500 images (250 PNG, 200 JPG, 50 BMP)

Step 1: Batch resize to maximum 1600x1200
[Calls batch_resize]
✓ Resized 500 images

Step 2: Batch compress to 85% quality
[Calls batch_compress with quality=85]
✓ Compressed 500 images

Step 3: Batch convert to WebP
[Calls batch_convert with format="webp"]
✓ Converted 500 images to WebP

Summary:
- Original: 1.8 GB (mixed formats)
- CDN-ready: 185 MB (WebP)
- Savings: 1.615 GB (89.7%)
- CDN bandwidth reduction: ~10x
- Estimated cost savings: $50-100/month
```

### Example 13: Archive Compression Without Quality Loss

**User Prompt:**
```
I have 1000 scanned documents (PNG format, 300 DPI) in 'scans' folder. Compress them for archival storage while maintaining visual quality.
```

**Claude's Approach:**
```
For archival documents, I'll use high-quality compression (95%) to maintain readability.

[Calls batch_compress with quality=95, workers=8]

✓ Batch compression complete
Processed: 1000 images
Original: 8.5 GB
Compressed: 2.1 GB
Savings: 6.4 GB (75.3%)

All documents maintain 300 DPI and readability. The compression removes redundant data without affecting text clarity.
```

---

## Web Development Use Cases

### Example 14: Generate Hero Images for Landing Page

**User Prompt:**
```
I'm building a SaaS landing page for a project management tool. Generate 3 hero images:
1. Clean, modern office with laptop showing dashboard
2. Team collaboration scene with people working together
3. Success/achievement visualization with growth charts

All should be 1792x1024 (wide format) and photorealistic.
```

**Claude's Execution:**
```
I'll generate three professional hero images for your SaaS landing page.

[Calls generate_image three times with detailed prompts and size="1792x1024"]

Generated images:
1. hero-office.png - Modern workspace with laptop dashboard
2. hero-collaboration.png - Diverse team in collaborative environment
3. hero-success.png - Growth charts and achievement visualization

All images are 1792x1024 (wide format), perfect for hero sections. They're photorealistic and professional, suitable for B2B SaaS marketing.
```

### Example 15: Create OpenGraph Social Preview Images

**User Prompt:**
```
I have 20 blog posts that need OpenGraph preview images. Requirements:
- Size: 1200x630 (OpenGraph standard)
- Text overlay space (will add text in design tool)
- Relevant to tech/programming topics
- Consistent visual style
```

**Claude's Workflow:**
```
I'll generate a set of OpenGraph images for your blog posts with consistent styling.

[Generates 20 images with prompts focused on tech, programming, abstract patterns]

For example:
- Post 1: "Abstract code visualization, blue and purple gradient"
- Post 2: "Minimalist circuit board pattern, technology theme"
- Post 3: "Digital network nodes, modern tech aesthetic"
...

✓ Generated 20 OpenGraph images (1200x630)
Style: Consistent modern tech aesthetic
Color palette: Blues, purples, grays
Text-overlay ready: Yes (prompts specified no text)

All images follow OpenGraph standards and will display correctly on Twitter, LinkedIn, Facebook, and Slack.
```

### Example 16: Generate SVG Placeholders for Lazy Loading

**User Prompt:**
```
I have 50 product images. Create tiny placeholder versions for lazy loading:
- 20x20 pixels (very small)
- High compression
- Will be used as blurred placeholders
```

**Claude's Implementation:**
```
I'll create ultra-small placeholders for your lazy loading implementation.

[Calls batch_resize with width=20, height=20]
[Calls batch_compress with quality=60]

✓ Created 50 placeholder images
Size: 20x20 pixels
Average file size: 1-2 KB each
Total: 75 KB for all 50 images

Usage in HTML:
<img src="placeholder-tiny.jpg"
     data-src="full-size-image.jpg"
     class="blur-up lazyload" />

These placeholders will load instantly while full images lazy-load, providing a smooth UX.
```

---

## E-commerce Workflows

### Example 17: Product Photography Pipeline

**User Prompt:**
```
I'm launching an online store with 100 products. For each product image (currently 6000x6000 PNG):
1. Create large view: 2000x2000 JPG
2. Create grid thumbnail: 600x600 JPG
3. Create cart thumbnail: 200x200 JPG
4. All at 90% quality

Images are in 'product-photos' folder.
```

**Claude's Complete Pipeline:**
```
I'll create a complete product image pipeline for your e-commerce store.

Processing 100 product images with 3 sizes each (300 total output images)...

Step 1: Create large views (2000x2000)
[Calls batch_resize to 'products-large' folder]
✓ Created 100 large views

Step 2: Create grid thumbnails (600x600)
[Calls batch_resize to 'products-grid' folder]
✓ Created 100 grid thumbnails

Step 3: Create cart thumbnails (200x200)
[Calls batch_resize to 'products-cart' folder]
✓ Created 100 cart thumbnails

Step 4: Compress all with 90% quality
[Calls batch_compress on each folder]
✓ Compressed all 300 images

Results:
- products-large/ (100 images, 45 MB total, avg 450 KB)
- products-grid/ (100 images, 8 MB total, avg 80 KB)
- products-cart/ (100 images, 1.2 MB total, avg 12 KB)

Total output: 300 images, 54.2 MB
Original: 100 images, 3.6 GB
Savings: 3.55 GB (98.5% reduction)

Your product catalog is now optimized for fast e-commerce page loads.
```

### Example 18: Seasonal Promotional Graphics

**User Prompt:**
```
Generate promotional graphics for our summer sale:
- 5 different beach/summer themed images
- 1024x1024 square format
- Bright, cheerful, summery vibes
- No text (we'll add text overlays)
```

**Claude's Creative Process:**
```
I'll generate 5 summer sale promotional graphics for you.

[Generates 5 images with summer themes, prompts specify no text overlays]

Generated images:
1. summer-promo-beach.png - Tropical beach with palm trees
2. summer-promo-pool.png - Refreshing pool scene with floaties
3. summer-promo-fruits.png - Colorful summer fruits arrangement
4. summer-promo-sunset.png - Golden hour beach sunset
5. summer-promo-tropical.png - Lush tropical foliage

All images:
- 1024x1024 square (perfect for social media)
- No text (ready for overlay design)
- Bright, saturated colors (summer aesthetic)
- Consistent style across all 5

Ready for use in social posts, email campaigns, and banner ads.
```

### Example 19: Product Variant Image Generation

**User Prompt:**
```
We're launching a new water bottle in 5 colors. Generate product mockups:
- Sleek modern water bottle
- Colors: Ocean Blue, Forest Green, Sunset Orange, Rose Pink, Charcoal Black
- White background
- Professional product photography style
- 1024x1024
```

**Claude's Product Generation:**
```
I'll generate professional product mockups for your water bottle in all 5 colors.

[Generates 5 images with specific color requirements]

Generated product mockups:
1. bottle-ocean-blue.png
2. bottle-forest-green.png
3. bottle-sunset-orange.png
4. bottle-rose-pink.png
5. bottle-charcoal-black.png

All images feature:
- Professional product photography lighting
- Clean white background (easy to cut out if needed)
- Consistent angle and composition
- 1024x1024 format

These mockups are ready for your product pages, catalog, and marketing materials.
```

---

## Social Media Content

### Example 20: Instagram Content Calendar

**User Prompt:**
```
Create a week's worth of Instagram content (7 posts) for a fitness brand:
- Monday: Motivation (sunrise workout scene)
- Tuesday: Nutrition (healthy meal prep)
- Wednesday: Workout (gym equipment)
- Thursday: Rest day (meditation/yoga)
- Friday: Challenge (intense training)
- Saturday: Community (group fitness)
- Sunday: Reflection (peaceful nature)

All 1080x1080, vibrant colors, inspirational mood.
```

**Claude's Content Strategy:**
```
I'll generate your complete Instagram content calendar for the week.

[Generates 7 themed images with consistent style]

Week's Content:
1. monday-motivation.png - Sunrise yoga on mountain peak
2. tuesday-nutrition.png - Colorful meal prep containers
3. wednesday-workout.png - Modern gym with equipment
4. thursday-rest.png - Serene meditation space
5. friday-challenge.png - Intense CrossFit training scene
6. saturday-community.png - Diverse group fitness class
7. sunday-reflection.png - Peaceful forest trail

All images:
- 1080x1080 Instagram square format
- Consistent vibrant color palette
- No text overlays (ready for captions)
- Professional, aspirational quality

Ready to schedule with your caption copy and hashtags.
```

### Example 21: LinkedIn Carousel Post Graphics

**User Prompt:**
```
Generate 5 images for a LinkedIn carousel about "5 Tips for Remote Work Productivity":
1. Home office setup
2. Time management concept
3. Communication/video calls
4. Work-life balance
5. Success/achievement

1080x1080, professional but approachable, modern office aesthetic.
```

**Claude's Carousel Creation:**
```
I'll create a professional carousel series for your LinkedIn post.

[Generates 5 images following the tip sequence]

Carousel Graphics:
1. slide-1-home-office.png - Modern, organized home workspace
2. slide-2-time-management.png - Clock and productivity tools
3. slide-3-communication.png - Professional video call setup
4. slide-4-work-life-balance.png - Balanced lifestyle visualization
5. slide-5-success.png - Achievement and growth representation

Series features:
- Consistent professional aesthetic
- Modern office/tech theme throughout
- 1080x1080 LinkedIn-optimized
- Visual flow between slides
- No text (you'll add tip overlays)

Perfect for engaging LinkedIn audience with actionable productivity content.
```

### Example 22: YouTube Thumbnail Generation

**User Prompt:**
```
Generate YouTube thumbnails for my tech review channel. I need 10 different backgrounds:
- Tech-themed
- High contrast colors (good for thumbnails)
- 1280x720 (YouTube standard)
- Space for face cutout and text overlay
- Variety of colors for different video topics
```

**Claude's Thumbnail Set:**
```
I'll generate 10 high-contrast tech backgrounds for your YouTube thumbnails.

[Generates 10 images with varied color schemes, all tech-themed]

Thumbnail Backgrounds:
1. tech-bg-red.png - Red circuit pattern
2. tech-bg-blue.png - Blue digital grid
3. tech-bg-green.png - Green code matrix
4. tech-bg-purple.png - Purple neural network
5. tech-bg-orange.png - Orange tech waves
6. tech-bg-cyan.png - Cyan data streams
7. tech-bg-yellow.png - Yellow chip design
8. tech-bg-pink.png - Pink tech gradients
9. tech-bg-teal.png - Teal digital mesh
10. tech-bg-gradient.png - Multi-color tech abstract

All thumbnails:
- 1280x720 YouTube standard
- High contrast for visibility
- Space reserved for face and text
- Distinct colors for easy browsing

These will make your videos stand out in YouTube search and suggested videos.
```

---

## Creative Content Generation

### Example 23: Book Cover Concept Variations

**User Prompt:**
```
I'm writing a sci-fi novel called "The Last Algorithm". Generate 3 cover concept variations:
1. Minimalist abstract (geometric shapes, AI theme)
2. Dramatic scene (futuristic city, lone figure)
3. Symbolic (digital consciousness visualization)

All 1024x1792 (portrait format for book cover).
```

**Claude's Design Concepts:**
```
I'll generate three distinct cover concepts for your sci-fi novel.

[Generates 3 images with different artistic approaches]

Cover Concepts:
1. cover-minimalist.png
   - Clean geometric shapes forming circuit pattern
   - Monochromatic with accent color
   - Modern, sophisticated aesthetic
   - Great for tech-savvy audience

2. cover-dramatic.png
   - Dystopian future city skyline
   - Lone figure silhouette
   - Moody, atmospheric lighting
   - Appeals to traditional sci-fi fans

3. cover-symbolic.png
   - Abstract digital consciousness
   - Neural network visualization
   - Ethereal, thought-provoking
   - Unique, artistic approach

All 1024x1792 portrait format. Each concept offers different marketing angles. You can test with your target audience or show to publishers.
```

### Example 24: Game Asset Generation

**User Prompt:**
```
I'm developing an indie game. Generate environment concept art:
- 5 different biomes: Forest, Desert, Ice, Volcano, Alien Planet
- 1024x1024 each
- Stylized art style (not photorealistic)
- Vibrant colors
- Top-down perspective
```

**Claude's Game Asset Creation:**
```
I'll generate 5 biome concepts for your indie game.

[Generates 5 images with stylized, game-art aesthetic]

Biome Concept Art:
1. biome-forest.png - Lush green forest with ancient trees
2. biome-desert.png - Golden dunes with rock formations
3. biome-ice.png - Frozen tundra with ice crystals
4. biome-volcano.png - Molten lava flows with ember particles
5. biome-alien.png - Otherworldly landscape with strange flora

Art style:
- Stylized, not photorealistic
- Vibrant, saturated colors
- Game-ready aesthetic
- Top-down/isometric perspective
- Consistent art direction

These concepts can guide your 3D modelers or be used directly in 2D game development.
```

### Example 25: Album Cover Art

**User Prompt:**
```
Generate album cover art for an electronic music album titled "Neon Dreams". I want:
- Cyberpunk/synthwave aesthetic
- Neon colors (pink, purple, blue)
- Abstract or architectural elements
- 3000x3000 (high-res for printing)
- Artistic, not literal
```

**Claude's Music Art Generation:**
```
I'll create album cover art with a cyberpunk synthwave aesthetic for "Neon Dreams".

[Generates high-resolution artwork with specified aesthetic]

Generated: neon-dreams-cover.png

Artwork features:
- Cyberpunk cityscape with neon reflections
- Dominant pink, purple, and blue palette
- Abstract geometric overlays
- Synthwave aesthetic with retro-future vibes
- 3000x3000 resolution (print-ready)

This cover will work for:
- Spotify/Apple Music (digital)
- Vinyl pressing (high resolution)
- Promotional materials
- Merchandise design

The abstract nature allows your music to define the interpretation, while the neon aesthetic immediately communicates the electronic genre.
```

---

## Advanced Multi-Step Workflows

### Example 26: Complete Website Image Pipeline

**User Prompt:**
```
I'm launching a complete website redesign. Process my image library:

Source: 'website-images' folder (250 mixed images)

Requirements:
1. Hero images: Resize to 1920x1080, WebP, 85% quality → 'assets/heroes'
2. Content images: Resize to 1200x800, WebP, 85% quality → 'assets/content'
3. Thumbnails: Resize to 400x300, WebP, 80% quality → 'assets/thumbs'
4. Icons/small: Resize to 64x64, PNG (need transparency) → 'assets/icons'

Only process JPG/PNG from source. Ignore everything else.
```

**Claude's Enterprise Pipeline:**
```
I'll process your complete image library for the website redesign.

Analyzing source folder...
Found: 250 images (180 JPG, 70 PNG)

Pipeline execution:

Step 1: Identify and copy hero-worthy images (landscape, high-res)
Found: 30 hero candidates

[Batch resize to 1920x1080]
[Batch convert to WebP]
[Batch compress to 85%]
✓ Processed 30 hero images → assets/heroes/

Step 2: Process standard content images
Found: 180 content images

[Batch resize to 1200x800]
[Batch convert to WebP]
[Batch compress to 85%]
✓ Processed 180 content images → assets/content/

Step 3: Generate thumbnails for all
[Batch resize to 400x300]
[Batch convert to WebP]
[Batch compress to 80%]
✓ Processed 250 thumbnails → assets/thumbs/

Step 4: Process icon candidates (square images only)
Found: 40 square images suitable for icons

[Batch resize to 64x64]
[Keep as PNG for transparency]
✓ Processed 40 icons → assets/icons/

Complete Pipeline Results:
────────────────────────────────────
Total images processed: 500 output files
Original size: 2.8 GB
Optimized size: 195 MB
Savings: 2.605 GB (93.0%)

Breakdown by type:
- Heroes: 30 images, 42 MB (1.4 MB avg)
- Content: 180 images, 108 MB (600 KB avg)
- Thumbnails: 250 images, 38 MB (152 KB avg)
- Icons: 40 images, 7 MB (175 KB avg)

Your website image library is now fully optimized and organized!
```

### Example 27: Marketing Campaign Asset Generation

**User Prompt:**
```
I need a complete asset package for our product launch campaign:

Generate:
1. 5 social media posts (1080x1080) - product in lifestyle settings
2. 3 email header images (600x200) - minimalist product shots
3. 1 billboard concept (4000x2000) - dramatic hero shot
4. 2 web banner ads (728x90) - product with CTA space

Product: Wireless headphones, modern design, colors: black and silver
Theme: Premium audio experience, lifestyle focus
```

**Claude's Campaign Asset Creation:**
```
I'll generate your complete marketing campaign asset package.

Campaign: Wireless Headphones Launch
Theme: Premium Audio Lifestyle

──────────────────────────────────
SOCIAL MEDIA POSTS (1080x1080)
──────────────────────────────────

[Generates 5 lifestyle images]

1. social-workout.png - Headphones during gym workout
2. social-commute.png - Urban commuter with headphones
3. social-study.png - Focused student studying with headphones
4. social-travel.png - Traveler with headphones at airport
5. social-relax.png - Relaxing at home with headphones

All images: Square format, lifestyle focus, product prominent but natural

──────────────────────────────────
EMAIL HEADERS (600x200)
──────────────────────────────────

[Generates 3 minimalist product shots]

1. email-hero-1.png - Black headphones on white, clean
2. email-hero-2.png - Silver headphones with subtle shadow
3. email-hero-3.png - Product detail shot, premium feel

All images: Wide format, minimalist, professional, email-safe

──────────────────────────────────
BILLBOARD CONCEPT (4000x2000)
──────────────────────────────────

[Generates high-res billboard image]

billboard-hero.png - Dramatic product shot with urban background
- High resolution for printing
- Bold composition
- Space for headline text
- Attention-grabbing from distance

──────────────────────────────────
WEB BANNER ADS (728x90)
──────────────────────────────────

[Generates 2 banner ads]

1. banner-ad-1.png - Product left, CTA space right
2. banner-ad-2.png - Product right, CTA space left

Both banners: Standard IAB format, clear CTA space, brand colors

──────────────────────────────────
CAMPAIGN ASSET SUMMARY
──────────────────────────────────

Total assets: 11 images
Formats: 4 different sizes
Theme: Cohesive premium lifestyle
Color palette: Black, silver, urban tones

All assets maintain consistent branding and visual language. Ready for:
- Social media scheduling
- Email marketing campaigns
- Display advertising
- Out-of-home advertising

Estimated campaign reach: Multi-channel coverage
```

### Example 28: AI-Generated Product Photography Workflow

**User Prompt:**
```
I can't afford a product photographer. Generate professional product photos for my handmade candle business:

Products:
- Soy candles in glass jars
- Colors: Amber, white, sage green
- Labels: Minimalist kraft paper

Shots needed:
- 3 hero shots (single candle, dramatic lighting)
- 2 lifestyle shots (candles in home settings)
- 1 flat lay (candles with natural props)
- Size: 1024x1024 for all

Style: Natural, organic, artisan aesthetic
```

**Claude's Product Photography Generation:**
```
I'll generate a complete professional product photography set for your candle business.

──────────────────────────────────
HERO SHOTS (Product Focus)
──────────────────────────────────

[Generates 3 hero product images]

1. hero-amber.png
   - Amber soy candle in glass jar
   - Dramatic side lighting
   - Dark background, warm glow
   - Kraft paper label visible
   - Professional studio quality

2. hero-white.png
   - White soy candle, clean aesthetic
   - Soft diffused lighting
   - Light neutral background
   - Minimalist composition
   - Artisan craftsmanship evident

3. hero-sage.png
   - Sage green candle
   - Natural window lighting effect
   - Organic, earthy vibe
   - Product details clear

──────────────────────────────────
LIFESTYLE SHOTS (In Context)
──────────────────────────────────

[Generates 2 lifestyle images]

1. lifestyle-living-room.png
   - Candles on coffee table
   - Cozy home interior
   - Soft ambient lighting
   - Natural home setting
   - Inviting atmosphere

2. lifestyle-bathroom.png
   - Candle near bathtub
   - Spa-like environment
   - Self-care theme
   - Calm, relaxing mood
   - Product in use context

──────────────────────────────────
FLAT LAY (Storytelling Shot)
──────────────────────────────────

[Generates flat lay composition]

flatlay-natural.png
   - Top-down view
   - Multiple candles arranged
   - Natural props (eucalyptus, stones)
   - Kraft paper, twine elements
   - Instagram-worthy composition
   - Organic, artisan aesthetic

──────────────────────────────────
PRODUCT PHOTOGRAPHY SET SUMMARY
──────────────────────────────────

Total shots: 6 professional images
All 1024x1024 (perfect for product pages and social media)
Style: Consistent natural, artisan aesthetic
Quality: Professional photography standard

Cost savings vs. hiring photographer: $500-1000

These images are ready for:
- Etsy product listings
- Instagram marketing
- Website product pages
- Social media content
- Email marketing
- Print catalogs

Your handmade candle business now has professional-looking product photography without the cost of a photographer or studio!
```

---

## Tips for Effective Prompts

### Be Specific
```
❌ "Generate an image of a forest"
✓ "Generate a photorealistic image of a misty pine forest at dawn with sunbeams filtering through trees"
```

### Specify Technical Requirements
```
❌ "Make it smaller"
✓ "Resize to 800x600 pixels for web use"
```

### State Exclusions in the Prompt
```
✓ "Generate a street scene with no people, no cars, and no modern buildings"
```

### Chain Operations Clearly
```
✓ "First resize to 1024x1024, then compress to 85% quality, then convert to WebP"
```

### Provide Context
```
✓ "I need hero images for a SaaS landing page targeting B2B customers"
```

---

## Next Steps

- Explore the [Complete Tools Reference](MCP_TOOLS.md)
- Read the [Usage Guide](MCP_USAGE.md)
- Try combining operations for your specific workflows
- Share your successful prompts with the community

## Support

- **GitHub Issues**: https://github.com/apresai/gimage/issues
- **Documentation**: https://github.com/apresai/gimage
