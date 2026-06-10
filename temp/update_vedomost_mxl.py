# -*- coding: utf-8 -*-
import re
from pathlib import Path

path = Path(r"E:\tl\src\cf\DataProcessors\GDS_ПечатьЗапровочныхВедомостей\Templates\Ведомость\Ext\Template.xml")
text = path.read_text(encoding="utf-8-sig")


def map_col(c):
    c = int(c)
    if c <= 2:
        return c
    if c <= 5:
        return c
    if c <= 8:
        return c + 2
    if c <= 10:
        return c + 2
    return c + 5


new_columns = """\t<columns>
\t\t<id>13c4a67e-b303-4241-90ef-ae2b4c4e0379</id>
\t\t<size>21</size>
\t\t<columnsItem><index>0</index><column><formatIndex>1</formatIndex></column></columnsItem>
\t\t<columnsItem><index>1</index><column><formatIndex>2</formatIndex></column></columnsItem>
\t\t<columnsItem><index>2</index><column><formatIndex>3</formatIndex></column></columnsItem>
\t\t<columnsItem><index>3</index><column><formatIndex>129</formatIndex></column></columnsItem>
\t\t<columnsItem><index>4</index><column><formatIndex>129</formatIndex></column></columnsItem>
\t\t<columnsItem><index>5</index><column><formatIndex>129</formatIndex></column></columnsItem>
\t\t<columnsItem><index>6</index><column><formatIndex>129</formatIndex></column></columnsItem>
\t\t<columnsItem><index>7</index><column><formatIndex>129</formatIndex></column></columnsItem>
\t\t<columnsItem><index>8</index><column><formatIndex>3</formatIndex></column></columnsItem>
\t\t<columnsItem><index>9</index><column><formatIndex>7</formatIndex></column></columnsItem>
\t\t<columnsItem><index>10</index><column><formatIndex>8</formatIndex></column></columnsItem>
\t\t<columnsItem><index>11</index><column><formatIndex>130</formatIndex></column></columnsItem>
\t\t<columnsItem><index>12</index><column><formatIndex>130</formatIndex></column></columnsItem>
\t\t<columnsItem><index>13</index><column><formatIndex>130</formatIndex></column></columnsItem>
\t\t<columnsItem><index>14</index><column><formatIndex>130</formatIndex></column></columnsItem>
\t\t<columnsItem><index>15</index><column><formatIndex>130</formatIndex></column></columnsItem>
\t\t<columnsItem><index>16</index><column><formatIndex>11</formatIndex></column></columnsItem>
\t\t<columnsItem><index>17</index><column><formatIndex>12</formatIndex></column></columnsItem>
\t\t<columnsItem><index>18</index><column><formatIndex>13</formatIndex></column></columnsItem>
\t\t<columnsItem><index>19</index><column><formatIndex>14</formatIndex></column></columnsItem>
\t\t<columnsItem><index>20</index><column><formatIndex>15</formatIndex></column></columnsItem>
\t</columns>"""

start = text.find("\t<columns>\n\t\t<id>13c4a67e")
end = text.find("</columns>", start) + len("</columns>")
text = text[:start] + new_columns + text[end:]

text = re.sub(r"<i>(\d+)</i>", lambda m: f"<i>{map_col(m.group(1))}</i>", text)


def repl_merge(m):
    r, c = m.group(1), int(m.group(2))
    h, w = m.group(3), m.group(4)
    nc = map_col(c)
    out = ["\t<merge>", f"\t\t<r>{r}</r>", f"\t\t<c>{nc}</c>"]
    if h:
        out.append(f"\t\t<h>{h}</h>")
    if w:
        nw = map_col(c + int(w)) - nc
        out.append(f"\t\t<w>{nw}</w>")
    out.append("\t</merge>")
    return "\n".join(out)


text = re.sub(
    r"\t<merge>\s*<r>(\d+)</r>\s*<c>(\d+)</c>(?:\s*<h>(\d+)</h>)?(?:\s*<w>(\d+)</w>)?\s*</merge>",
    repl_merge,
    text,
    flags=re.S,
)

text = re.sub(r"\t<merge>\s*<r>9</r>.*?</merge>\s*", "", text, flags=re.S)

if "Dotted" not in text:
    text = text.replace(
        '\t<line width="1" gap="false">\n\t\t<v8ui:style xsi:type="v8ui:SpreadsheetDocumentCellLineType">Solid</v8ui:style>\n\t</line>',
        '\t<line width="1" gap="false">\n\t\t<v8ui:style xsi:type="v8ui:SpreadsheetDocumentCellLineType">Solid</v8ui:style>\n\t</line>\n\t<line width="1" gap="false">\n\t\t<v8ui:style xsi:type="v8ui:SpreadsheetDocumentCellLineType">Dotted</v8ui:style>\n\t</line>',
        1,
    )

formats = """
\t<format><width>55</width></format>
\t<format><width>38</width></format>
\t<format><font>0</font><leftBorder>2</leftBorder><topBorder>2</topBorder><bottomBorder>2</bottomBorder><borderColor>#000000</borderColor><width>55</width><horizontalAlignment>Center</horizontalAlignment><verticalAlignment>Center</verticalAlignment><textPlacement>Cut</textPlacement></format>
\t<format><font>0</font><topBorder>2</topBorder><bottomBorder>2</bottomBorder><borderColor>#000000</borderColor><width>55</width><horizontalAlignment>Center</horizontalAlignment><verticalAlignment>Center</verticalAlignment><textPlacement>Cut</textPlacement></format>
\t<format><font>0</font><topBorder>2</topBorder><rightBorder>2</rightBorder><bottomBorder>2</bottomBorder><borderColor>#000000</borderColor><width>55</width><horizontalAlignment>Center</horizontalAlignment><verticalAlignment>Center</verticalAlignment><textPlacement>Cut</textPlacement></format>
\t<format><font>0</font><leftBorder>2</leftBorder><topBorder>2</topBorder><bottomBorder>2</bottomBorder><borderColor>#000000</borderColor><width>38</width><horizontalAlignment>Center</horizontalAlignment><verticalAlignment>Center</verticalAlignment><textPlacement>Cut</textPlacement></format>
\t<format><font>0</font><topBorder>2</topBorder><bottomBorder>2</bottomBorder><borderColor>#000000</borderColor><width>38</width><horizontalAlignment>Center</horizontalAlignment><verticalAlignment>Center</verticalAlignment><textPlacement>Cut</textPlacement></format>
\t<format><font>0</font><topBorder>2</topBorder><rightBorder>2</rightBorder><bottomBorder>2</bottomBorder><borderColor>#000000</borderColor><width>38</width><horizontalAlignment>Center</horizontalAlignment><verticalAlignment>Center</verticalAlignment><textPlacement>Cut</textPlacement></format>"""

text = text.replace("\t<picture>", formats + "\n\t<picture>", 1)

row9 = """\t<rowsItem>
\t\t<index>9</index>
\t\t<row>
\t\t\t<columnsID>13c4a67e-b303-4241-90ef-ae2b4c4e0379</columnsID>
\t\t\t<formatIndex>29</formatIndex>
\t\t\t<c><c><f>90</f><parameter>НомерСтроки</parameter></c></c>
\t\t\t<c><c><f>91</f></c></c>
\t\t\t<c><c><f>91</f></c></c>
\t\t\t<c><c><f>131</f></c></c>
\t\t\t<c><c><f>132</f></c></c>
\t\t\t<c><c><f>132</f></c></c>
\t\t\t<c><c><f>132</f></c></c>
\t\t\t<c><c><f>133</f></c></c>
\t\t\t<c><c><f>91</f></c></c>
\t\t\t<c><c><f>93</f></c></c>
\t\t\t<c><c><f>93</f></c></c>
\t\t\t<c><c><f>134</f></c></c>
\t\t\t<c><c><f>135</f></c></c>
\t\t\t<c><c><f>135</f></c></c>
\t\t\t<c><c><f>135</f></c></c>
\t\t\t<c><c><f>136</f></c></c>
\t\t\t<c><c><f>92</f></c></c>
\t\t\t<c><c><f>95</f></c></c>
\t\t\t<c><c><f>96</f></c></c>
\t\t\t<c><c><f>91</f></c></c>
\t\t\t<c><c><f>91</f></c></c>
\t\t</row>
\t</rowsItem>"""

text = re.sub(r"\t<rowsItem>\s*<index>9</index>.*?</rowsItem>", row9, text, count=1, flags=re.S)
text = re.sub(r"(<merge>\s*<r>12</r>\s*<c>0</c>\s*<w>)14(</w>)", r"\g<1>19\2", text)

path.write_text(text, encoding="utf-8")
print("OK")
