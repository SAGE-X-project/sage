#!/usr/bin/env python3
"""
Mermaid 다이어그램 추출 및 PNG 변환 스크립트
"""

import re
import os

# 입력/출력 경로
base_dir = "/Users/kevin/work/github/sage-x-project/sage/docs/final-report"
input_file = f"{base_dir}/report-with-visuals.md"
output_dir = f"{base_dir}/assets/mermaid-source"
image_dir = f"{base_dir}/assets/diagrams"

# 마크다운 파일 읽기
with open(input_file, 'r', encoding='utf-8') as f:
    content = f.read()

# Mermaid 코드 블록 패턴
pattern = r'```mermaid\n(.*?)\n```'
matches = re.findall(pattern, content, re.DOTALL)

print(f"총 {len(matches)}개의 Mermaid 다이어그램을 찾았습니다.")

# 각 다이어그램을 파일로 저장
for i, mermaid_code in enumerate(matches, 1):
    # .mmd 파일로 저장
    mmd_filename = f"{output_dir}/diagram-{i:02d}.mmd"
    with open(mmd_filename, 'w', encoding='utf-8') as f:
        f.write(mermaid_code)
    print(f"생성됨: {mmd_filename}")

print(f"\n모든 Mermaid 다이어그램이 {output_dir}/ 디렉토리에 저장되었습니다.")
print(f"다음 단계: mmdc로 PNG 변환")
