import json,sys,collections,os
from pathlib import Path
sys.path.insert(0,'/tmp/sage-analysis-deps')
from pglast import parser
O=Path('analysis/2026-09-12');G=O/'graphs'
def save(p,x):p.write_text(json.dumps(x,ensure_ascii=False,indent=2)+'\n')
c=json.loads((G/'code-index.json').read_text());sql=[]
for f in c['files']:
 if f['language']!='sql':continue
 try:
  ast=json.loads(parser.parse_sql_json(Path(f['file']).read_text()));sql.append({'file':f['file'],'ast':ast});f['alternative_parser']='pglast 7.7 / PostgreSQL';f['alternative_status']='parsed';f['status']='parsed_postgresql'
 except Exception as e:f['alternative_status']='failed';f['alternative_error']=str(e)
save(G/'postgresql-ast.json',sql);save(G/'code-index.json',c)
coverage=json.loads((O/'evidence/coverage.json').read_text());coverage['parse_status']=dict(collections.Counter(f['status'] for f in c['files']));coverage['sql_note']='Tree-sitter SQL dialect errors retained; PostgreSQL grammar reparses all 4 files. PL/pgSQL function bodies remain string literals in PostgreSQL statement AST.';save(O/'evidence/coverage.json',coverage)
# Native Go semantic graph uses go/packages which also found ignored directories.
# Preserve raw output and make an explicitly inventory-filtered graph.
g=json.loads((G/'go-semantic/graph.json').read_text());snap=json.loads((O/'evidence/snapshot.json').read_text());fs={x['path'] for x in snap['files']};mod=g['module']
symbols=[s for s in g['symbols'] if s['file'] in fs];pkgset={s['package'] for s in symbols}|{mod+'/'+str(Path(f).parent) for f in fs if f.endswith('.go')};ps=[p for p in g['packages'] if p['path'] in pkgset];ids={s['id'] for s in symbols}|{p['path'] for p in ps}
edges=[e for e in g['edges'] if e['from'] in ids and e['to'] in ids]
filtered={**g,'packages':ps,'symbols':symbols,'edges':edges,'commands':[x for x in g.get('commands',[]) if x['file'] in fs]};save(G/'go-semantic/graph-filtered.json',filtered)
save(O/'evidence/go-graph-filter.json',{'raw_packages':len(g['packages']),'filtered_packages':len(ps),'raw_symbols':len(g['symbols']),'filtered_symbols':len(symbols),'raw_edges':len(g['edges']),'filtered_edges':len(edges),'removed_packages':[p['path'] for p in g['packages'] if p['path'] not in pkgset],'note':'Filtered to original snapshot inventory; go/packages excludes some build-tagged files and nested modules. All source files still have syntax ASTs.'})
# Candidate links only; NEVER infer implemented/missing from lexical matches.
bs=json.loads((G/'document-blocks.json').read_text());cand=[];cedges=[]
import re
pat=re.compile(r'\b(must|shall|should|guarantee[sd]?|prevent[sd]?|support[sd]?|provide[sd]?|implement[sd]?|required|TODO)\b|보장|방지|지원|구현|필수|미구현|미지원|해야',re.I)
for b in bs:
 if b['kind'] in ('code_fence','image_reference') or not pat.search(b['text']):continue
 names=set(re.findall(r'\b[A-Za-z][A-Za-z0-9_]{4,}\b',b['text']));matches=[s for s in symbols if s['name'] in names]
 cid='candidate:'+b['id'];cand.append({'id':cid,'block':b['id'],'file':b['file'],'line':b['line'],'text':b['text'],'status':'unadjudicated','symbols':[s['id'] for s in matches[:30]],'link_method':'exact_identifier_lexical_match; not semantic entailment'})
 for s in matches[:30]:cedges.append({'source':cid,'target':s['id'],'kind':'lexical_symbol_candidate'})
save(G/'claim-candidates.json',{'candidates':cand,'edges':cedges});print('SQL',[(f['file'],f['status']) for f in c['files'] if f['language']=='sql']);print('Go',len(ps),len(symbols),len(edges),'claim candidates',len(cand))
