#!/usr/bin/env python3
"""Repository inventory, all-language named syntax trees and document block graph.
Run from repository root. No source files are modified. NLP is in syntax_docs.py.
"""
import collections, hashlib, json, os, re, subprocess, sys, gzip
from pathlib import Path
sys.path.insert(0, os.environ.get('SAGE_PARSER_DEPS','/tmp/sage-analysis-deps'))
from tree_sitter_language_pack import get_parser
ROOT=Path.cwd(); OUT=ROOT/'analysis/2026-09-12'; G=OUT/'graphs'; G.mkdir(parents=True,exist_ok=True)
def write(p,x): p.write_text(json.dumps(x,ensure_ascii=False,indent=2)+'\n')
files=[f for f in subprocess.check_output(['git','ls-files','--cached','--others','--exclude-standard','-z']).decode().split('\0') if f and not f.startswith('analysis/2026-09-12/') and Path(f).is_file()]
files=sorted(set(files)); write(OUT/'evidence/snapshot.json',{'commit':subprocess.check_output(['git','rev-parse','HEAD']).decode().strip(),'status':subprocess.check_output(['git','status','--short']).decode(),'files':[{'path':f,'sha256':hashlib.sha256(Path(f).read_bytes()).hexdigest(),'bytes':Path(f).stat().st_size} for f in files]})
langs={'.go':'go','.py':'python','.js':'javascript','.ts':'typescript','.tsx':'tsx','.java':'java','.rs':'rust','.sol':'solidity','.sh':'bash','.sql':'sql','.yml':'yaml','.yaml':'yaml','.json':'json','.toml':'toml','.xml':'xml'}
code=[]; nodes=[]; edges=[]; parsers={}; totals=collections.Counter(); full=gzip.open(G/'code-ast.jsonl.gz','wt',encoding='utf8')
for f in files:
 p=Path(f); lang=langs.get(p.suffix)
 if p.name=='Makefile': lang='make'
 if p.name.startswith('Dockerfile'): lang='dockerfile'
 if not lang: continue
 b=p.read_bytes(); errors=[]; count=0; declarations=[]
 try:
  parser=parsers.setdefault(lang,get_parser(lang)); tree=parser.parse(b)
  stack=[(tree.root_node,None)]; rootid='ast:'+f+':0'
  while stack:
   n,parent=stack.pop(); nid='ast:'+f+':'+str(count); count+=1
   row={'id':nid,'file':f,'kind':n.type,'start_byte':n.start_byte,'end_byte':n.end_byte,'line':n.start_point.row+1,'end_line':n.end_point.row+1,'parent':parent,'is_error':n.is_error,'is_missing':n.is_missing}
   if n.child_count==0: row['text']=b[n.start_byte:n.end_byte].decode('utf8','replace')[:240]
   full.write(json.dumps(row,ensure_ascii=False)+'\n')
   if n.is_error or n.is_missing: errors.append({'kind':n.type,'line':n.start_point.row+1,'text':b[n.start_byte:n.end_byte].decode('utf8','replace')[:140]})
   if any(s in n.type for s in ('function_declaration','function_definition','method_declaration','class_declaration','class_definition','struct_item','trait_item','interface_declaration','contract_declaration','function_item','impl_item','type_declaration')):
    name=n.child_by_field_name('name'); label=(name.text.decode('utf8','replace') if name else n.type)
    d={'id':nid,'kind':n.type,'label':label,'file':f,'line':n.start_point.row+1}; declarations.append(d);nodes.append(d);edges.append({'source':'file:'+f,'target':nid,'kind':'declares'})
   children=[(c,nid) for c in n.children if c.is_named or c.is_missing]; stack.extend(reversed(children))
  status='parsed_with_errors' if tree.root_node.has_error else 'parsed'
 except Exception as e: status='failed'; errors=[{'error':str(e)}]
 nodes.append({'id':'file:'+f,'label':f,'kind':'file','language':lang}); code.append({'file':f,'language':lang,'status':status,'nodes':count,'errors':errors,'declarations':len(declarations),'generated':('bindings/' in f or 'flattened/' in f),'test':bool(re.search(r'(^|/)(test|tests)/|_test\.go$|\.test\.',f))});totals[lang]+=count
full.close();write(G/'code-index.json',{'files':code,'nodes':nodes,'edges':edges,'ast_format':'Each JSONL row is a named Tree-sitter syntax node. parent defines child_of edges. Punctuation omitted, byte spans retained. Not a resolved call graph.','totals':dict(totals)})
# Documents: preserve blocks and source lines; taxonomy is explicitly lexical, not syntax.
D=[]; N=[]; E=[]; blocks=[]
topics={'identity':r'\bdid\b|identity|신원|registry|agentcard','handshake':r'hpke|handshake|핸드셰이크|핸드쉐이크|키 교환','integrity':r'rfc.?9421|signature|서명|무결성|replay|재전송|nonce','session':r'session|세션|rekey|rotation','transport':r'\bmcp\b|\ba2a\b|websocket|transport|전송','operations':r'deploy|배포|ci/cd|monitor|모니터|docker','governance':r'license|licens|governance|라이선스|거버넌스','validation':r'test|테스트|benchmark|성능|검증','architecture':r'architect|아키텍처|refactor|리팩터|design|설계'}
for f in files:
 p=Path(f)
 if p.suffix not in ('.md','.txt') and p.name not in ('LICENSE','NOTICE','VERSION'): continue
 text=p.read_text(errors='replace'); fid='doc:'+f
 state='archived' if '/archive/' in f else ('historical_assessment' if '/refactoring/analysis/' in f or p.name in ('FEATURE_MAP.md','SECURITY_WIRING_AUDIT.md','DOCS_GRAPH.md') else 'unversioned')
 topics_hit=[k for k,v in topics.items() if re.search(v,text,re.I)]
 N.append({'id':fid,'kind':'document','label':f,'state':state,'topics':topics_hit}); D.append({'file':f,'state':state,'topics':topics_hit})
 section=fid; sections=[]; lines=text.splitlines(); i=0
 while i<len(lines):
  if not lines[i].strip(): i+=1;continue
  start=i; raw=lines[i]; kind='paragraph'
  m=re.match(r'^(#{1,6})\s+(.+)',raw)
  if m:
   kind='heading';depth=len(m[1]);
   while sections and sections[-1][0]>=depth: sections.pop()
   parent=sections[-1][1] if sections else fid;i+=1
  elif re.match(r'^\s*(```|~~~)',raw):
   kind='code_fence';marker=raw.strip()[:3];i+=1
   while i<len(lines) and not lines[i].strip().startswith(marker): i+=1
   i=min(i+1,len(lines));parent=section
  elif raw.lstrip().startswith('|'): kind='table_row';i+=1;parent=section
  elif re.match(r'^\s*(?:[-*+] |\d+[.)] )',raw): kind='list_item';i+=1;parent=section
  elif '<img' in raw or re.match(r'^\s*!\[',raw): kind='image_reference';i+=1;parent=section
  else:
   i+=1;parent=section
   while i<len(lines) and lines[i].strip() and not re.match(r'^\s*(?:#|```|~~~|\||[-*+] |\d+[.)] |!\[|<img)',lines[i]):i+=1
  raw='\n'.join(lines[start:i]);bid=f'{fid}:L{start+1}'; row={'id':bid,'file':f,'line':start+1,'end_line':i,'kind':kind,'text':raw,'parent':parent}
  blocks.append(row);N.append({k:v for k,v in row.items() if k!='text'}|{'label':raw[:140]});E.append({'source':parent,'target':bid,'kind':'contains'})
  if kind=='heading': sections.append((depth,bid));section=bid
  for target in re.findall(r'\[[^\]]*\]\(([^ )]+)',raw):
   clean=target.split('#')[0]
   if clean and not re.match(r'\w+://|mailto:',clean):
    resolved=os.path.normpath(str(p.parent/clean));E.append({'source':bid,'target':('doc:' if Path(resolved).suffix=='.md' else 'file:')+resolved,'kind':'references','exists':Path(resolved).exists()})
 for topic in topics_hit:E.append({'source':fid,'target':'topic:'+topic,'kind':'lexical_topic'})
for topic in topics:N.append({'id':'topic:'+topic,'kind':'topic','label':topic})
write(G/'document-structure.json',{'documents':D,'nodes':N,'edges':E});write(G/'document-blocks.json',blocks)
# Explicitly classify remaining tracked files rather than silently skipping them.
covered={x['file'] for x in code}|{x['file'] for x in D}
write(OUT/'evidence/coverage.json',{'inventory':len(files),'documents':len(D),'source_and_config':len(code),'by_language':dict(collections.Counter(x['language'] for x in code)),'parse_status':dict(collections.Counter(x['status'] for x in code)),'other_files':[{'file':f,'kind':'image' if Path(f).suffix=='.png' else 'metadata_or_extensionless_config'} for f in files if f not in covered],'ast_named_nodes':sum(totals.values()),'document_blocks':len(blocks)})
print(json.dumps(json.loads((OUT/'evidence/coverage.json').read_text()),ensure_ascii=False))
