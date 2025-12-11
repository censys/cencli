package prompts

const BasePrompt = `
	Using the supplied data, create a high-fidelity visualization for marketing material that is striking and dramatic.

	CRITICAL LAYOUT REQUIREMENTS:
	- Generate exactly ONE card containing the visualization. Do NOT create a multi-card dashboard layout.
	- The card should be dominated by the main visualization.
	- Any supplementary statistics (totals, top items, etc.) should appear as subtle annotations WITHIN the card, not as separate cards or sidebars.
	- Annotations should be minimal and placed in corners or along edges so they don't obstruct the main visualization.

	REQUIRED ELEMENTS (must appear on every visualization):
	- TITLE: A clear, bold title at the top of the card describing the data
	- SUBTITLE: A descriptive subtitle below the title explaining the aggregation field used
	- QUERY: Display the exact Censys query string used to generate this data, formatted in a monospace font at the bottom of the card

	STYLE GUIDE:
	- Modern minimalist SaaS aesthetic in "light mode" with clean layout and generous white space.
	- Single pure white card (#FFFFFF) with subtle 1px light gray border (#E2E2E2) against pale gray background (#F9FAFB).
	- Typography is clean sans-serif: dark charcoal (#333333) for primary values, muted cool gray (#757575) for labels.
	- Primary data is emphasized using soft apricot (#F4C495).
	- Label accents use muted slate teal (#4F8A96).
	- Do not include any interactive components such as buttons or filters.

	Wherever possible, include a small, simple icon next to each category label that visually conveys the meaning or theme of the category.
	Icons should be:
	- Minimalist and monochrome (e.g., neutral gray)
	- Small enough not to distract, but large enough to be recognizable
	- Consistent in style, weight, and alignment
	`

// ChartPrompts contains chart-specific prompt fragments that can be added on top of the base prompt.
// Each chart type has its own prompt fragment that describes how to visualize the data.
// Use the chart type constants (e.g., ChartTypeGeographicMap) as keys.
var ChartPrompts = map[string]string{
	"geomap": `
		Represent the geographic data as 3D pillars on an accurate stylized map.
		Pillar heights should be proportional to the data values.
		Each pillar should have a label showing the location name and value.
		Data pillars use soft apricot (#F4C495).
		The 3D map should be highly stylized and accurate, with the geographic region clearly recognizable.
		`,
	"voronoi": `
		Represent the data as a Voronoi chart. The visualization should be minimalist.
		Do not include any visual elements that distract from the data. Include a subtle
		gradient that emphasizes the data.
		`,
	"pie": `
		Represent the data as a pie chart. The visualization should be minimalist.
		Do not include any visual elements that distract from the data.
	`,
	"wordmap": `
		Represent the data as a word map. More frequently appearing words should be larger.
		The visualization should be minimalist.
		Do not include any visual elements that distract from the data.
	`,
	"choropleth": `
		Represent the geographic data as a clean, minimalist choropleth map. Color all regions using a gradient from light to dark soft apricot (#F4C495), where darker shades correspond to higher values. Include clear, accurate geographic boundaries appropriate to the dataset (e.g., countries, provinces, states, or custom regions).
		Label every region using its short region code (e.g., ISO country code, state/province code, or other provided identifier). Place each code unobtrusively within or near its region.
		Highlight only the top N regions by value (e.g., the top 5–10):
		- add a second label showing the numeric value near the region code,
		- optionally increase border weight or add a subtle outline for emphasis,
		- ensure highlighted labels are placed cleanly and do not overlap.
		Include a minimalist legend showing the value range and the color gradient.
		The visualization should be accurate, proportionally scaled, and stylistically minimal, with smooth color transitions, crisp boundaries, and no unnecessary elements.
	`,
	"hextile": `
		Represent the country data as a tile chart instead of a traditional map.
		Each state should be shown as a uniform tile of identical shape and size (for example, a simple square), with no original geographic boundaries or irregular state shapes.
		Arrange the tiles in a layout that roughly follows the geographic position of the states across the US, but keep the grid/tile structure clean and consistent.
		Use soft apricot (#F4C495) for the tiles, with darker shades for higher values.
		Include state labels and values on or near each tile.
		The visualization should be minimalist and modern.
	`,
	"globe": `
		Represent the geographic exposure data as a clean, high-resolution 3D globe.
		Use a country-level choropleth, where each country is shaded using a smooth gradient from soft apricot (#F4C495) to darker tones for higher exposure values.
		Keep the shading bounded to each country (avoid diffuse blobs or exaggerated heatmaps).
		Orient the globe so that major regions with significant exposure (e.g., North America, Europe, Asia) are clearly visible.
		Draw thin, minimalist country outlines to maintain geographic clarity.
		Include subtle, well-placed labels for key countries and regions, ensuring they do not overlap or obscure shaded areas.
		Use soft ambient lighting and a modern aesthetic without visual clutter.
		Avoid glowing artifacts, random graph elements, or distortions.
		The final visualization should feel clean, minimalist, and information-rich.
	`,
	"bubble": `
		Represent the data as a bubble chart using uniformly styled circular bubbles.
		Each bubble's area (not radius) should scale proportionally to its value to maintain accurate visual comparison.
		Arrange the bubbles using a non-overlapping layout (e.g., force-directed packing) that preserves a clear visual hierarchy, with larger bubbles naturally drawing more attention. Keep spacing even and avoid visual clutter.
		Style each bubble in soft apricot (#F4C495) with subtle transparency and a thin, neutral outline for definition.
		Place legible labels inside or near each bubble showing the key and value, ensuring labels do not overlap and remain readable regardless of bubble size.
		Use a minimalist, clean aesthetic: no gridlines, axes, or unnecessary decorations—focus on clarity, balance, and modern visual design.
	`,
	"bar": `
		Represent the data as a vertical bar chart, using only the top 8–10 categories ranked by value.
		Bars must be sorted in strict descending order, with the highest-value category on the far left and the lowest on the right.
		Scale each bar's height proportionally to its value. Fill bars with soft apricot (#F4C495) and use a thin, neutral outline for definition.
		For each category, display a small minimalist icon above or next to the label to visually reinforce the meaning of the category. Icons should be
		- simple and monochrome,
		- consistent in size and stroke weight,
		- unobtrusive and aligned cleanly with the label.
		Include clear, readable labels for each category at the bottom of each bar, and show numeric values either at the top of the bars or immediately above them. Labels must not overlap.
		Use a minimalist layout—no heavy gridlines, no axis clutter, no unnecessary decorations. Maintain generous spacing between bars and balanced margins so the chart feels clean, modern, and easy to interpret.
	`,
	"smallmultiplesbar": `
		Represent the geographic data as a set of small multiples.
		Create a uniform grid of small charts, where each chart corresponds to one geographic region (country or state).
		Use a consistent visual encoding across all multiples—such as a single vertical bar, a compact area sparkline, or a filled color-intensity tile—to represent that region's data value. The encoding must be identical in style, scale, and orientation for every region to allow direct comparison.
		Apply soft apricot (#F4C495) as the primary encoding color across all panels, using darker or more saturated tones only when necessary to indicate higher values.
		Each region's multiple should include:
		- a clear region label (e.g., the country or state name),
		- the data value, placed unobtrusively but legibly,
		- consistent margins and spacing so that every panel aligns cleanly.
		Arrange the multiples in a logical, structured grid—for example grouped by continent, subregion, or alphabetical order. Ensure the grid remains balanced and easy to scan.
		Keep the overall design minimalist and modern, with no heavy borders, excessive text, or redundant axes. Only include minimal visual cues necessary to compare values across regions.
	`,
	"smallmultiplesmap": `
		Represent the geographic data as small multiples with mini geographic maps.
		Create a uniform grid of small tiles, where each tile contains a simplified outline of the country or state, rendered as a small geographic shape.
		Inside each geographic outline, apply a choropleth-style fill using soft apricot (#F4C495), with darker or more saturated tones indicating higher values. The shading should stay within the country boundary, not extend outside it.
		All country outlines must use the same visual style: thin, neutral strokes, simplified geometry, and consistent scaling so that shapes remain recognizable but comparable across tiles.
		Each tile must include:
		- a country/region label,
		- the data value, placed cleanly below or beside the mini-map,
		- consistent padding and alignment.
		Arrange all tiles in a logical grid layout, grouped by continent or subregion when applicable.
		Keep the overall design minimalist and modern—no icons, no bars, no heavy borders, no unnecessary UI elements. The primary focus should be the geographic mini-map and its choropleth shading, enabling quick visual comparison between regions.
	`,
}

// ChartTypes returns all available chart type names.
func ChartTypes() []string {
	types := make([]string, 0, len(ChartPrompts))
	for k := range ChartPrompts {
		types = append(types, k)
	}
	return types
}
