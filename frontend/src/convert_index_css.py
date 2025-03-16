import re
import math
from typing import Tuple

def oklch_to_oklab(l: float, c: float, h: float) -> Tuple[float, float, float]:
    h_rad = math.radians(h)
    a = c * math.cos(h_rad)
    b = c * math.sin(h_rad)
    return l, a, b

def oklab_to_linear_srgb(l: float, a: float, b: float) -> Tuple[float, float, float]:
    l_ = l + 0.3963377774 * a + 0.2158037573 * b
    m_ = l - 0.1055613458 * a - 0.0638541728 * b
    s_ = l - 0.0894841775 * a - 1.2914855480 * b

    l = l_ * l_ * l_
    m = m_ * m_ * m_
    s = s_ * s_ * s_

    r = +4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s
    g = -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s
    b = -0.0041960863 * l - 0.7034186147 * m + 1.7076147010 * s

    return r, g, b

def linear_to_srgb(x: float) -> float:
    if x <= 0:
        return 0
    elif x >= 1:
        return 1
    elif x <= 0.0031308:
        return x * 12.92
    else:
        return 1.055 * math.pow(x, 1/2.4) - 0.055

def rgb_to_hsl(r: float, g: float, b: float) -> Tuple[float, float, float]:
    r = linear_to_srgb(r)
    g = linear_to_srgb(g)
    b = linear_to_srgb(b)
    
    max_val = max(r, g, b)
    min_val = min(r, g, b)
    
    h = 0
    s = 0
    l = (max_val + min_val) / 2

    if max_val != min_val:
        d = max_val - min_val
        s = d / (2 - max_val - min_val) if l > 0.5 else d / (max_val + min_val)
        
        if max_val == r:
            h = (g - b) / d + (6 if g < b else 0)
        elif max_val == g:
            h = (b - r) / d + 2
        elif max_val == b:
            h = (r - g) / d + 4
        
        h /= 6

    # 提高精度，保留一位小数
    return round(h * 360, 1), round(s * 100, 1), round(l * 100, 1)

def oklch_to_hsl(l: float, c: float, h: float) -> Tuple[float, float, float]:
    lab_l, lab_a, lab_b = oklch_to_oklab(l, c, h)
    rgb_r, rgb_g, rgb_b = oklab_to_linear_srgb(lab_l, lab_a, lab_b)
    return rgb_to_hsl(rgb_r, rgb_g, rgb_b)

def convert_css_colors(css_text: str) -> str:
    oklch_pattern = r'oklch\((\d*\.?\d+)\s+(\d*\.?\d+)\s+(\d*\.?\d+)(?:\s*\/\s*(\d*\.?\d+%?))?\)'
    
    def replace_color(match):
        l = float(match.group(1))
        c = float(match.group(2))
        h = float(match.group(3))
        alpha = match.group(4)
        
        hsl_h, hsl_s, hsl_l = oklch_to_hsl(l, c, h)
        
        if alpha:
            return f'hsla({hsl_h}, {hsl_s}%, {hsl_l}%, {alpha})'
        return f'hsl({hsl_h}, {hsl_s}%, {hsl_l}%)'
    
    return re.sub(oklch_pattern, replace_color, css_text)

def main():
    try:
        with open('index.css', 'r') as file:
            css_text = file.read()
    except FileNotFoundError:
        print("Please create an index.css file with your CSS content")
        return

    converted_css = convert_css_colors(css_text)

    with open('output.css', 'w') as file:
        file.write(converted_css)
    
    print("Conversion completed! Check output.css for the results.")

if __name__ == "__main__":
    main()