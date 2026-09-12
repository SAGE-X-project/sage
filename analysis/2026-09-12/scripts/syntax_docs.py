#!/usr/bin/env python3
"""Parse every eligible natural-language block offline; UD -> explicit X-bar-inspired projection.
This is not a constituency parser or a claim of linguistically validated X-bar trees.
"""
import collections,gzip,hashlib,json,os,re,time
from pathlib import Path
import stanza,torch
OUT=Path('analysis/2026-09-12');G=OUT/'graphs'
torch.set_num_threads(2)
blocks=json.loads((G/'document-blocks.json').read_text());groups={'en':{},'ko':{}};coverage=[]
def clean(s):
 s=re.sub(r'!\[[^\]]*\]\([^)]*\)','',s);s=re.sub(r'\[([^\]]*)\]\([^)]*\)',r'\1',s)
 s=re.sub(r'https?://\S+','URL',s);s=re.sub(r'<[^>]+>',' ',s)
 return re.sub(r'\s+',' ',s.strip(' #>*|-`')).strip()
for b in blocks:
 row={'block':b['id'],'file':b['file'],'line':b['line'],'syntax_ids':[]}
 if b['kind'] in ('code_fence','image_reference'):row['status']='non_prose';coverage.append(row);continue
 s=clean(b['text'])
 if len(re.findall(r'[A-Za-z가-힣]+',s))<2:row['status']='non_linguistic_fragment';coverage.append(row);continue
 # Keep bounded fragments; every character of normalized eligible text is included.
 parts=[s[i:i+1200] for i in range(0,len(s),1200)]
 for part in parts:
  lang='ko' if re.search('[가-힣]',part) else 'en';sid='syntax:'+hashlib.sha256((lang+part).encode()).hexdigest()[:20]
  row['syntax_ids'].append(sid);groups[lang][sid]=part
 row['status']='queued';row['fragmented']=len(parts)>1;coverage.append(row)
( G/'syntax-coverage.json').write_text(json.dumps(coverage,ensure_ascii=False))
print('unique', {k:len(v) for k,v in groups.items()},flush=True)
# X-bar-inspired projection: each UD lexical head gets XP, X', X0.
# Complement and adjunct ordering is based on observed token position. Subject/det roles
# are coarse hypotheses; no silent nodes, movement, theta grids, or CP/TP reconstruction.
cat={'NOUN':'N','PROPN':'N','PRON':'N','VERB':'V','AUX':'T','ADJ':'A','ADV':'Adv','ADP':'P','DET':'D','SCONJ':'C','CCONJ':'Conj','NUM':'Num','PART':'Part','PUNCT':'Punct'}
stats=collections.Counter();failures=[];parsed=set();sample=[]
for lang,items in groups.items():
 pkg='ewt' if lang=='en' else 'kaist'
 nlp=stanza.Pipeline(lang,dir='/tmp/sage-stanza',package=None,processors={'tokenize':pkg,'pos':pkg+'_nocharlm','lemma':pkg+'_nocharlm','depparse':pkg+'_nocharlm'},download_method=None,use_gpu=False,verbose=False)
 entries=list(items.items())
 with gzip.open(G/f'document-syntax-{lang}.jsonl.gz','wt',encoding='utf8') as out:
  for start in range(0,len(entries),32):
   batch=entries[start:start+32]
   try:docs=nlp.bulk_process([s for _,s in batch])
   except Exception as e:
    failures.extend({'id':sid,'error':str(e)} for sid,_ in batch);continue
   for (sid,text),doc in zip(batch,docs):
    sentences=[]
    for si,sent in enumerate(doc.sentences):
     tokens=[];nodes=[];edges=[];base=f'{sid}:s{si}'
     for w in sent.words:
      t=f'{base}:t{w.id}';c=cat.get(w.upos,'X');xp=t+':XP';bar=t+":bar";head=t+':head'
      tokens.append({'id':w.id,'text':w.text,'lemma':w.lemma,'upos':w.upos,'xpos':w.xpos,'head':w.head,'deprel':w.deprel})
      nodes.extend([{'id':xp,'label':c+'P','kind':'projection'},{'id':bar,'label':c+"′",'kind':'bar'},{'id':head,'label':w.text,'category':c,'kind':'head'}]);edges.extend([{'source':xp,'target':bar,'kind':'projects'},{'source':bar,'target':head,'kind':'head'}])
      if w.head:
       rel=w.deprel.split(':')[0];role='specifier_candidate' if rel in ('nsubj','csubj','det') else ('complement_candidate' if rel in ('obj','iobj','ccomp','xcomp') else ('punctuation' if rel=='punct' else 'adjunct_or_functional_candidate'))
       parent=f'{base}:t{w.head}'+(':XP' if role=='specifier_candidate' else ':bar')
       edges.append({'source':parent,'target':xp,'kind':role,'ud_relation':w.deprel,'order':'before' if w.id<w.head else 'after','confidence':'heuristic_projection'})
     sentences.append({'id':base,'text':sent.text,'tokens':tokens,'nodes':nodes,'edges':edges});stats['sentences']+=1;stats['tokens']+=len(tokens);stats['xbar_nodes']+=len(nodes);stats['xbar_edges']+=len(edges)
    row={'id':sid,'language':lang,'text':text,'sentences':sentences,'method':'Stanza UD neural parse + X-bar-inspired non-binary projection; not validated constituency'}
    out.write(json.dumps(row,ensure_ascii=False)+'\n');parsed.add(sid)
    if len(sample)<20 or ('replay' in text.lower() and len(sample)<35):sample.append(row)
   if start%320==0: print(lang,start,'/',len(entries),'sentences',stats['sentences'],flush=True)
 del nlp
for r in coverage:
 if r['status']=='queued':r['status']='parsed' if all(x in parsed for x in r['syntax_ids']) else 'failed'
(G/'syntax-coverage.json').write_text(json.dumps(coverage,ensure_ascii=False))
(G/'syntax-sample.json').write_text(json.dumps(sample,ensure_ascii=False))
(G/'syntax-summary.json').write_text(json.dumps({'counts':dict(stats),'unique_fragments':len(parsed),'block_status':dict(collections.Counter(r['status'] for r in coverage)),'failures':failures,'models':{'en':'ewt_nocharlm','ko':'kaist_nocharlm'},'stanza_version':stanza.__version__,'limitations':['UD to X-bar is non-unique; roles are heuristic, not a gold constituency analysis','Head-final Korean particles and functional projections are not reconstructed','Tables, headings and snippets are fragments; no sentence-level truth or security conclusion is inferred automatically','Source line range is attached at block level; 1200-character fragments may split a sentence']},ensure_ascii=False,indent=2))
print('DONE',dict(stats),flush=True)
