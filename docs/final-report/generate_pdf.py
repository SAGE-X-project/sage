#!/usr/bin/env python3
"""
Markdown을 HTML로 변환하고 PDF 생성
"""

import markdown
from weasyprint import HTML, CSS
from weasyprint.text.fonts import FontConfiguration
import os

# 경로
base_dir = "/Users/kevin/work/github/sage-x-project/sage/docs/final-report"
input_file = f"{base_dir}/report-with-images.md"
output_html = f"{base_dir}/report-with-images.html"
output_pdf = f"{base_dir}/SAGE-Final-Report.pdf"

print("1. 마크다운 파일 읽기...")
with open(input_file, 'r', encoding='utf-8') as f:
    md_content = f.read()

print("2. Markdown → HTML 변환...")
md = markdown.Markdown(extensions=[
    'extra',
    'codehilite',
    'tables',
    'toc'
])
html_body = md.convert(md_content)

# HTML 템플릿
html_template = f"""<!DOCTYPE html>
<html lang="ko">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SAGE - Secure Agent Guarantee Engine</title>
    <style>
        @page {{
            size: A4;
            margin: 2cm;
        }}

        body {{
            font-family: 'Noto Sans KR', 'Malgun Gothic', sans-serif;
            line-height: 1.8;
            color: #343A40;
            max-width: 800px;
            margin: 0 auto;
        }}

        h1 {{
            color: #4DABF7;
            font-size: 28pt;
            border-bottom: 3px solid #4DABF7;
            padding-bottom: 10px;
            margin-top: 30px;
            page-break-after: avoid;
        }}

        h2 {{
            color: #4DABF7;
            font-size: 22pt;
            margin-top: 25px;
            page-break-after: avoid;
        }}

        h3 {{
            color: #343A40;
            font-size: 18pt;
            margin-top: 20px;
            page-break-after: avoid;
        }}

        h4 {{
            color: #343A40;
            font-size: 16pt;
            margin-top: 15px;
        }}

        p {{
            margin: 10px 0;
            text-align: justify;
        }}

        img {{
            max-width: 100%;
            height: auto;
            display: block;
            margin: 20px auto;
            page-break-inside: avoid;
        }}

        code {{
            background-color: #F1F3F5;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: 'Consolas', 'Monaco', monospace;
            font-size: 0.9em;
        }}

        pre {{
            background-color: #F1F3F5;
            padding: 15px;
            border-radius: 5px;
            overflow-x: auto;
            page-break-inside: avoid;
        }}

        pre code {{
            background: none;
            padding: 0;
        }}

        table {{
            border-collapse: collapse;
            width: 100%;
            margin: 20px 0;
            page-break-inside: avoid;
        }}

        table th {{
            background-color: #4DABF7;
            color: white;
            padding: 10px;
            text-align: left;
            font-weight: bold;
        }}

        table td {{
            border: 1px solid #DEE2E6;
            padding: 10px;
        }}

        table tr:nth-child(even) {{
            background-color: #F8F9FA;
        }}

        blockquote {{
            border-left: 4px solid #4DABF7;
            margin: 20px 0;
            padding-left: 20px;
            color: #495057;
            font-style: italic;
        }}

        ul, ol {{
            margin: 10px 0;
            padding-left: 30px;
        }}

        li {{
            margin: 5px 0;
        }}

        hr {{
            border: none;
            border-top: 2px solid #DEE2E6;
            margin: 30px 0;
        }}

        .page-break {{
            page-break-before: always;
        }}

        /* 링크 */
        a {{
            color: #4DABF7;
            text-decoration: none;
        }}

        a:hover {{
            text-decoration: underline;
        }}

        /* 강조 */
        strong {{
            color: #343A40;
            font-weight: bold;
        }}

        em {{
            font-style: italic;
        }}
    </style>
</head>
<body>
{html_body}
</body>
</html>
"""

print("3. HTML 파일 저장...")
with open(output_html, 'w', encoding='utf-8') as f:
    f.write(html_template)

print(f"   → {output_html}")

print("4. HTML → PDF 변환...")
# WeasyPrint로 PDF 생성
font_config = FontConfiguration()

# CSS 추가 (필요시)
css = CSS(string='''
    @page {
        size: A4;
        margin: 2cm;
        @bottom-center {
            content: counter(page) " / " counter(pages);
            font-size: 10pt;
            color: #868E96;
        }
    }
''', font_config=font_config)

HTML(output_html).write_pdf(output_pdf, stylesheets=[css], font_config=font_config)

print(f"   → {output_pdf}")

print("\n✅ 완료!")
print(f"\n생성된 파일:")
print(f"  - HTML: {output_html}")
print(f"  - PDF:  {output_pdf}")

# 파일 크기 확인
html_size = os.path.getsize(output_html) / 1024 / 1024
pdf_size = os.path.getsize(output_pdf) / 1024 / 1024
print(f"\n파일 크기:")
print(f"  - HTML: {html_size:.2f} MB")
print(f"  - PDF:  {pdf_size:.2f} MB")
