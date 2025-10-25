#!/usr/bin/env python3
"""
Markdown을 PDF용 HTML로 변환 (브라우저 인쇄 최적화)
"""

import markdown
import os
import subprocess

# 경로
base_dir = "/Users/kevin/work/github/sage-x-project/sage/docs/final-report"
input_file = f"{base_dir}/report-with-images.md"
output_html = f"{base_dir}/SAGE-Final-Report.html"

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

# HTML 템플릿 (인쇄 최적화)
html_template = f"""<!DOCTYPE html>
<html lang="ko">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SAGE - Secure Agent Guarantee Engine - Final Report</title>
    <style>
        /* 인쇄용 스타일 */
        @media print {{
            @page {{
                size: A4;
                margin: 2cm;
            }}

            body {{
                margin: 0;
                padding: 0;
            }}

            h1, h2, h3, h4 {{
                page-break-after: avoid;
            }}

            img, table, pre {{
                page-break-inside: avoid;
            }}

            a {{
                color: #4DABF7;
                text-decoration: none;
            }}

            /* 페이지 번호 */
            @page {{
                @bottom-center {{
                    content: counter(page) " / " counter(pages);
                    font-size: 10pt;
                    color: #868E96;
                }}
            }}
        }}

        /* 화면 및 인쇄 공통 스타일 */
        body {{
            font-family: 'Noto Sans KR', 'Apple SD Gothic Neo', 'Malgun Gothic', sans-serif;
            line-height: 1.8;
            color: #343A40;
            max-width: 1000px;
            margin: 0 auto;
            padding: 20px;
            background-color: #FFFFFF;
        }}

        h1 {{
            color: #4DABF7;
            font-size: 32pt;
            border-bottom: 4px solid #4DABF7;
            padding-bottom: 12px;
            margin-top: 40px;
            margin-bottom: 20px;
        }}

        h2 {{
            color: #4DABF7;
            font-size: 24pt;
            margin-top: 30px;
            margin-bottom: 15px;
            border-bottom: 2px solid #E3F2FD;
            padding-bottom: 8px;
        }}

        h3 {{
            color: #343A40;
            font-size: 20pt;
            margin-top: 25px;
            margin-bottom: 12px;
        }}

        h4 {{
            color: #343A40;
            font-size: 16pt;
            margin-top: 20px;
            margin-bottom: 10px;
        }}

        p {{
            margin: 12px 0;
            text-align: justify;
            font-size: 11pt;
        }}

        img {{
            max-width: 100%;
            height: auto;
            display: block;
            margin: 25px auto;
            border: 1px solid #DEE2E6;
            border-radius: 8px;
            padding: 10px;
            background: #F8F9FA;
        }}

        code {{
            background-color: #F1F3F5;
            padding: 3px 7px;
            border-radius: 4px;
            font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
            font-size: 0.95em;
            color: #E64980;
        }}

        pre {{
            background-color: #F1F3F5;
            padding: 18px;
            border-radius: 6px;
            overflow-x: auto;
            border-left: 4px solid #4DABF7;
            margin: 20px 0;
        }}

        pre code {{
            background: none;
            padding: 0;
            color: #343A40;
        }}

        table {{
            border-collapse: collapse;
            width: 100%;
            margin: 25px 0;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }}

        table th {{
            background-color: #4DABF7;
            color: white;
            padding: 12px;
            text-align: left;
            font-weight: bold;
            font-size: 11pt;
        }}

        table td {{
            border: 1px solid #DEE2E6;
            padding: 10px;
            font-size: 10pt;
        }}

        table tr:nth-child(even) {{
            background-color: #F8F9FA;
        }}

        blockquote {{
            border-left: 5px solid #4DABF7;
            margin: 25px 0;
            padding: 15px 20px;
            background-color: #E3F2FD;
            color: #495057;
            font-style: italic;
            border-radius: 4px;
        }}

        ul, ol {{
            margin: 15px 0;
            padding-left: 35px;
        }}

        li {{
            margin: 8px 0;
            font-size: 11pt;
        }}

        hr {{
            border: none;
            border-top: 3px solid #DEE2E6;
            margin: 40px 0;
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
            color: #495057;
        }}

        /* 커버 페이지 */
        .cover-page {{
            text-align: center;
            padding: 100px 20px;
            page-break-after: always;
        }}

        .cover-page h1 {{
            font-size: 48pt;
            color: #4DABF7;
            margin-bottom: 20px;
            border: none;
        }}

        .cover-page .subtitle {{
            font-size: 24pt;
            color: #343A40;
            margin-bottom: 40px;
        }}

        .cover-page .info {{
            font-size: 16pt;
            color: #868E96;
            margin-top: 60px;
        }}

        /* 인쇄 버튼 (화면에만 표시) */
        .print-button {{
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 15px 30px;
            background-color: #4DABF7;
            color: white;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            font-size: 16pt;
            font-weight: bold;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            z-index: 1000;
        }}

        .print-button:hover {{
            background-color: #339AF0;
        }}

        @media print {{
            .print-button {{
                display: none;
            }}
        }}

        /* 목차 스타일 */
        .toc {{
            background-color: #F8F9FA;
            padding: 20px;
            border-radius: 8px;
            margin: 30px 0;
            border: 1px solid #DEE2E6;
        }}

        .toc h2 {{
            margin-top: 0;
            color: #4DABF7;
            border: none;
            padding-bottom: 0;
        }}

        .toc ul {{
            list-style-type: none;
            padding-left: 20px;
        }}

        .toc li {{
            margin: 8px 0;
        }}
    </style>
</head>
<body>
    <button class="print-button" onclick="window.print()">📄 PDF로 저장</button>

    <div class="cover-page">
        <h1>SAGE</h1>
        <div class="subtitle">Secure Agent Guarantee Engine</div>
        <div class="subtitle" style="font-size: 18pt; color: #4DABF7;">Trust Layer for AI Agent Era</div>
        <div class="info">
            <p>2025 오픈소스 개발자대회</p>
            <p>최종 발표 자료</p>
            <p style="margin-top: 40px;">SAGE-X Project Team</p>
        </div>
    </div>

{html_body}

    <script>
        // 자동 PDF 저장 안내
        window.addEventListener('load', function() {{
            const printButton = document.querySelector('.print-button');
            if (printButton) {{
                printButton.addEventListener('click', function() {{
                    alert('인쇄 대화상자에서:\\n1. "대상"을 "PDF로 저장" 선택\\n2. "저장" 버튼 클릭\\n\\n또는:\\nCmd+P (Mac) / Ctrl+P (Windows)');
                }});
            }}
        }});
    </script>
</body>
</html>
"""

print("3. HTML 파일 저장...")
with open(output_html, 'w', encoding='utf-8') as f:
    f.write(html_template)

print(f"   → {output_html}")

# 파일 크기 확인
html_size = os.path.getsize(output_html) / 1024 / 1024
print(f"\n파일 크기: {html_size:.2f} MB")

print("\n✅ HTML 파일 생성 완료!")
print("\n" + "="*60)
print("📄 PDF 생성 방법:")
print("="*60)
print("\n방법 1: 브라우저에서 직접 PDF 생성 (추천)")
print("  1. 생성된 HTML 파일을 더블클릭하여 브라우저에서 열기")
print("  2. 우측 상단 '📄 PDF로 저장' 버튼 클릭")
print("  3. 또는 Cmd+P (Mac) / Ctrl+P (Windows)")
print("  4. 대상: 'PDF로 저장' 선택")
print("  5. 저장 클릭")
print("\n방법 2: Chrome 명령줄로 PDF 생성")
print("  다음 명령어를 실행:")

# Chrome/Chromium 경로 찾기
chrome_paths = [
    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
    "/Applications/Chromium.app/Contents/MacOS/Chromium",
]

chrome_path = None
for path in chrome_paths:
    if os.path.exists(path):
        chrome_path = path
        break

if chrome_path:
    pdf_path = f"{base_dir}/SAGE-Final-Report.pdf"
    cmd = f'"{chrome_path}" --headless --disable-gpu --print-to-pdf="{pdf_path}" "file://{output_html}"'
    print(f"\n  {cmd}")
    print(f"\n자동 실행 중...")

    try:
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=30)
        if os.path.exists(pdf_path):
            pdf_size = os.path.getsize(pdf_path) / 1024 / 1024
            print(f"\n✅ PDF 생성 완료!")
            print(f"   → {pdf_path}")
            print(f"   크기: {pdf_size:.2f} MB")
        else:
            print("\n⚠️  PDF 자동 생성 실패. 브라우저에서 수동으로 생성해주세요.")
    except Exception as e:
        print(f"\n⚠️  PDF 자동 생성 실패: {e}")
        print("   브라우저에서 수동으로 생성해주세요.")
else:
    print("\n  (Chrome이 설치되어 있지 않습니다)")
    print("\n방법 3: 파일 브라우저에서")
    print(f"  {output_html}")
    print("  파일을 더블클릭하여 브라우저에서 열기 → Cmd+P → PDF로 저장")

print("\n" + "="*60)
