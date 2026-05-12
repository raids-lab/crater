"""轻量 drawio (.drawio XML) 生成器。

设计目标：在 Python 中以代码组织节点 / 连线 / 泳道 / 时序，统一应用
``styles/drawio_palette.md`` 中定义的字体与配色，避免手工 XML 散落造成的样式不一致。
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Iterable, List, Optional
from xml.etree.ElementTree import Element, SubElement, tostring


FONT_FAMILY = "Source Han Sans SC,Noto Sans CJK SC,PingFang SC,Hiragino Sans GB,sans-serif"


# 主题色（与 matplotlib 一致）
PALETTE = {
    "mops":       "#264653",
    "ps":         "#2A9D8F",
    "react":      "#9C89B8",
    "accent":     "#E76F51",
    "ok":         "#52B788",
    "warn":       "#F4A261",
    "danger":     "#C44536",
    "bg_lane":    "#F5F6F8",
    "bg_lane2":   "#EEF2F5",
    "bg_card":    "#FFFFFF",
    "border":     "#374151",
    "muted":      "#9CA3AF",
    "text":       "#1F2937",
    "subtle":     "#6B7280",
    "highlight":  "#F4E1D2",
    "light_blue": "#CFE3EA",
    "light_green":"#D9F2EC",
    "light_purple":"#E5DCF1",
}


def _style(items: dict) -> str:
    """构造 drawio 样式字符串。"""
    return ";".join(f"{k}={v}" for k, v in items.items() if v is not None) + ";"


def node_style(fill: str = "#FFFFFF", stroke: str = PALETTE["border"],
               text_color: str = PALETTE["text"], font_size: int = 9,
               font_style: int = 0, rounded: bool = True,
               shadow: bool = False, dashed: bool = False) -> str:
    items = {
        "shape": "rectangle",
        "rounded": "1" if rounded else "0",
        "arcSize": "6",
        "fillColor": fill,
        "strokeColor": stroke,
        "strokeWidth": "1.2",
        "fontFamily": FONT_FAMILY,
        "fontSize": str(font_size),
        "fontStyle": str(font_style),
        "fontColor": text_color,
        "align": "center",
        "verticalAlign": "middle",
        "spacingTop": "2",
        "spacingBottom": "2",
        "spacingLeft": "6",
        "spacingRight": "6",
        "html": "1",
    }
    if shadow:
        items["shadow"] = "1"
    if dashed:
        items["dashed"] = "1"
    return _style(items)


def lane_style(fill: str = PALETTE["bg_lane"], stroke: str = "#CBD5E1",
               title_size: int = 10) -> str:
    return _style({
        "shape": "swimlane",
        "startSize": "26",
        "horizontal": "1",
        "rounded": "1",
        "arcSize": "5",
        "fillColor": fill,
        "strokeColor": stroke,
        "fontFamily": FONT_FAMILY,
        "fontSize": str(title_size),
        "fontStyle": "1",
        "fontColor": PALETTE["text"],
        "align": "left",
        "verticalAlign": "top",
        "spacingLeft": "10",
        "html": "1",
    })


def edge_style(
    *,
    stroke: str = PALETTE["border"],
    dashed: bool = False,
    end_arrow: str = "classic",
    label_size: int = 8,
    orthogonal: bool = True,
    start_arrow: str = "none",
) -> str:
    items = {
        "endArrow": end_arrow,
        "endFill": "1",
        "startArrow": start_arrow,
        "html": "1",
        "fontFamily": FONT_FAMILY,
        "fontSize": str(label_size),
        "fontColor": PALETTE["text"],
        "strokeColor": stroke,
        "strokeWidth": "1.1",
        "edgeStyle": "orthogonalEdgeStyle" if orthogonal else "straightEdge",
        "rounded": "1" if orthogonal else "0",
        "jettySize": "auto",
        "exitX": None,
        "exitY": None,
        "entryX": None,
        "entryY": None,
    }
    if dashed:
        items["dashed"] = "1"
    return _style(items)


def actor_style() -> str:
    return _style({
        "shape": "umlActor",
        "verticalLabelPosition": "bottom",
        "verticalAlign": "top",
        "html": "1",
        "outlineConnect": "0",
        "fontFamily": FONT_FAMILY,
        "fontSize": "10",
        "fontStyle": "1",
        "fontColor": PALETTE["text"],
        "fillColor": PALETTE["mops"],
        "strokeColor": PALETTE["border"],
    })


def lifeline_style() -> str:
    return _style({
        "shape": "umlLifeline",
        "perimeter": "lifelinePerimeter",
        "container": "1",
        "collapsible": "0",
        "dropTarget": "0",
        "html": "1",
        "fontFamily": FONT_FAMILY,
        "fontSize": "9",
        "fontStyle": "1",
        "fontColor": PALETTE["text"],
        "fillColor": "#FFFFFF",
        "strokeColor": PALETTE["border"],
        "size": "30",
    })


def message_style(dashed: bool = False, async_msg: bool = False) -> str:
    return _style({
        "html": "1",
        "verticalAlign": "bottom",
        "endArrow": "classic" if not async_msg else "open",
        "endFill": "0" if async_msg else "1",
        "curved": "0",
        "rounded": "0",
        "strokeColor": PALETTE["border"],
        "strokeWidth": "1.0",
        "fontFamily": FONT_FAMILY,
        "fontSize": "9",
        "fontColor": PALETTE["text"],
        "dashed": "1" if dashed else "0",
    })


# -----------------------------------------------------------------------------
# Builder
# -----------------------------------------------------------------------------

@dataclass
class DrawioBuilder:
    name: str
    width: int = 1100
    height: int = 760
    _root: Element = field(init=False)
    _model: Element = field(init=False)
    _layer: Element = field(init=False)
    _next_id: int = 2

    def __post_init__(self):
        self._mxfile = Element("mxfile", attrib={
            "host": "app.diagrams.net",
            "modified": "2026-05-11T00:00:00.000Z",
            "agent": "docs-figure builder",
            "version": "24.0.0",
            "compressed": "false",
        })
        diagram = SubElement(self._mxfile, "diagram", attrib={"id": "main", "name": self.name})
        self._model = SubElement(diagram, "mxGraphModel", attrib={
            "dx": "1200", "dy": "800", "grid": "1", "gridSize": "10",
            "guides": "1", "tooltips": "1", "connect": "1", "arrows": "1",
            "fold": "1", "page": "1", "pageScale": "1",
            "pageWidth": str(self.width), "pageHeight": str(self.height),
            "math": "0", "shadow": "0",
        })
        self._root = SubElement(self._model, "root")
        SubElement(self._root, "mxCell", attrib={"id": "0"})
        SubElement(self._root, "mxCell", attrib={"id": "1", "parent": "0"})
        self._layer = "1"

    def _new_id(self) -> str:
        nid = str(self._next_id)
        self._next_id += 1
        return nid

    def node(self, text: str, x: int, y: int, w: int = 160, h: int = 40,
             style: Optional[str] = None, parent: str = "1") -> str:
        nid = self._new_id()
        cell = SubElement(self._root, "mxCell", attrib={
            "id": nid, "value": text,
            "style": style or node_style(),
            "vertex": "1", "parent": parent,
        })
        SubElement(cell, "mxGeometry", attrib={
            "x": str(x), "y": str(y), "width": str(w), "height": str(h),
            "as": "geometry",
        })
        return nid

    def lane(self, text: str, x: int, y: int, w: int, h: int,
             fill: str = PALETTE["bg_lane"], stroke: str = "#CBD5E1") -> str:
        nid = self._new_id()
        cell = SubElement(self._root, "mxCell", attrib={
            "id": nid, "value": text,
            "style": lane_style(fill, stroke),
            "vertex": "1", "parent": "1",
        })
        SubElement(cell, "mxGeometry", attrib={
            "x": str(x), "y": str(y), "width": str(w), "height": str(h),
            "as": "geometry",
        })
        return nid

    def edge(self, src: str, dst: str, label: str = "",
             style: Optional[str] = None,
             exit_xy: Optional[tuple] = None,
             entry_xy: Optional[tuple] = None) -> str:
        nid = self._new_id()
        s = style or edge_style()
        if exit_xy:
            s = s.replace("exitX=None", f"exitX={exit_xy[0]}").replace("exitY=None", f"exitY={exit_xy[1]}")
        else:
            s = s.replace("exitX=None;", "").replace("exitY=None;", "")
        if entry_xy:
            s = s.replace("entryX=None", f"entryX={entry_xy[0]}").replace("entryY=None", f"entryY={entry_xy[1]}")
        else:
            s = s.replace("entryX=None;", "").replace("entryY=None;", "")
        attrib = {
            "id": nid, "style": s,
            "edge": "1", "parent": "1",
            "source": src, "target": dst,
        }
        if label:
            attrib["value"] = label
        cell = SubElement(self._root, "mxCell", attrib=attrib)
        SubElement(cell, "mxGeometry", attrib={"relative": "1", "as": "geometry"})
        return nid

    def text_label(self, text: str, x: int, y: int, w: int = 120, h: int = 20,
                   font_size: int = 8, color: str = PALETTE["subtle"]) -> str:
        nid = self._new_id()
        style = _style({
            "text": "1", "html": "1", "strokeColor": "none", "fillColor": "none",
            "align": "left", "verticalAlign": "middle",
            "whiteSpace": "wrap", "rounded": "0",
            "fontFamily": FONT_FAMILY, "fontSize": str(font_size),
            "fontColor": color,
        })
        cell = SubElement(self._root, "mxCell", attrib={
            "id": nid, "value": text, "style": style, "vertex": "1", "parent": "1",
        })
        SubElement(cell, "mxGeometry", attrib={
            "x": str(x), "y": str(y), "width": str(w), "height": str(h),
            "as": "geometry",
        })
        return nid

    def write(self, out_path) -> None:
        from pathlib import Path
        out = Path(out_path)
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(tostring(self._mxfile, encoding="unicode"))
