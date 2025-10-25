#!/usr/bin/env python3
"""
Mermaid 코드 블록을 PNG 이미지 참조로 교체
"""

import re
import os

# 경로
base_dir = "/Users/kevin/work/github/sage-x-project/sage/docs/final-report"
input_file = f"{base_dir}/report-with-visuals.md"
output_file = f"{base_dir}/report-with-images.md"
image_dir = "assets/diagrams"

# 마크다운 파일 읽기
with open(input_file, 'r', encoding='utf-8') as f:
    content = f.read()

# Mermaid 코드 블록 패턴
pattern = r'```mermaid\n(.*?)\n```'

# 카운터
counter = 1

def replace_mermaid(match):
    global counter

    # PNG 파일 경로
    png_path = f"{image_dir}/diagram-{counter:02d}.png"
    full_path = f"{base_dir}/{png_path}"

    # PNG 파일이 존재하는지 확인
    if os.path.exists(full_path):
        replacement = f'![Diagram {counter}]({png_path})'
        print(f"교체 {counter}: {png_path}")
    else:
        # PNG 파일이 없으면 원본 유지
        replacement = match.group(0)
        print(f"유지 {counter}: PNG 파일 없음")

    counter += 1
    return replacement

# 교체 실행
new_content = re.sub(pattern, replace_mermaid, content, flags=re.DOTALL)

# 새 파일로 저장
with open(output_file, 'w', encoding='utf-8') as f:
    f.write(new_content)

print(f"\n완료! 새 파일 생성: {output_file}")
print(f"총 {counter-1}개의 Mermaid 블록 처리")
