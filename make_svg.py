centers = []
for line in open('E:/tl/dots.txt'):
    x, y = map(float, line.strip().split(','))
    centers.append((x, y))
svg = ['<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 522 1038" width="522" height="1038">']
for x, y in centers:
    svg.append(f'  <circle cx="{x:.1f}" cy="{y:.1f}" r="10" fill="black" />')
svg.append('</svg>')
open('E:/tl/dots.svg', 'w').write('\n'.join(svg))
