/**
 * 该代码为 https://github.com/teralomaniac/clewd 中的片段
 */

'use strict';

const ROLE_PREFIXS = {
    user: 'Human',
    assistant: 'Assistant',
    example_user: 'H',
    example_assistant: 'A',
    system: 'SYSTEM'
}, HyperProcess = (system, messages, claudeMode) => {
    const hyperMerge = (content, mergeDisable) => {
        let splitContent = content.split(new RegExp(`\\n\\n(${ROLE_PREFIXS['assistant']}|${ROLE_PREFIXS['user']}|${ROLE_PREFIXS['system']}):`, 'g'));
        content = splitContent[0] + splitContent.slice(1).reduce((acc, current, index, array) => {
            const merge = index > 1 && current === array[index - 2] && (
                current === ROLE_PREFIXS['user'] && !mergeDisable.user ||
                current === ROLE_PREFIXS['assistant'] && !mergeDisable.assistant ||
                current === ROLE_PREFIXS['system'] && !mergeDisable.system
            );
            return acc + (index % 2 !== 0 ? current.trim() : `\n\n${merge ? '' : `${current}: `}`)
        }, '');
        return content;
    }, hyperRegex = (content, order) => {
        let regexLog = '', matches = content.match(new RegExp(`<regex(?: +order *= *${order})${order === 2 ? '?' : ''}> *"(/?)(.*)\\1(.*?)" *: *"(.*?)" *</regex>`, 'gm'));
        matches && matches.forEach(match => {
            try {
                const reg = /<regex(?: +order *= *\d)?> *"(\/?)(.*)\1(.*?)" *: *"(.*?)" *<\/regex>/.exec(match);
                regexLog += match + '\n';
                if (reg[3].includes('s')) {
                    reg[2] = reg[2].replace(/([^\\])\./g, '$1[\\s\\S]')
                    reg[3] = reg[3].replace('s', 'm')
                }
                content = content.replace(new RegExp(reg[2], reg[3]), JSON.parse(`"${reg[4].replace(/\\?"/g, '\\"')}"`));
            } catch {}
        });
        return [content, regexLog];
    }, HyperPmtProcess = content => {
        const regex1 = hyperRegex(content, 1);
        content = regex1[0], regexLogs += regex1[1];
        const mergeDisable = {
            all: content.includes('<|Merge Disable|>'),
            system: content.includes('<|Merge System Disable|>'),
            user: content.includes('<|Merge Human Disable|>'),
            assistant: content.includes('<|Merge Assistant Disable|>')
        };
        content = content.replace(new RegExp(`(\\n\\n|^\\s*)(?<!\\n\\n(${ROLE_PREFIXS['user']}|${ROLE_PREFIXS['assistant']}):.*?)${ROLE_PREFIXS['system']}:\\s*`, 'gs'), '$1')
            .replace(new RegExp(`(\\n\\n|^\\s*)${ROLE_PREFIXS['system']}: *`, 'g'), mergeDisable.all || mergeDisable.user || mergeDisable.system ? '$1' : `\n\n${ROLE_PREFIXS['user']}: `);
        content = hyperMerge(content, mergeDisable);
        let splitContent = content.split(new RegExp(`\\n\\n(?=${ROLE_PREFIXS['assistant']}:|${ROLE_PREFIXS['user']}:)`, 'g')), match;
        while ((match = /<@(\d+)>(.*?)<\/@\1>/gs.exec(content)) !== null) {
            let index = splitContent.length - parseInt(match[1]) - 1;
            index >= 0 && (splitContent[index] += '\n\n' + match[2]);
            content = content.replace(match[0], '');
        }
        content = splitContent.join('\n\n').replace(/<@(\d+)>.*?<\/@\1>/gs, '');
        const regex2 = hyperRegex(content, 2);
        content = regex2[0], regexLogs += regex2[1];
        content = hyperMerge(content, mergeDisable);
        const regex3 = hyperRegex(content, 3);
        content = regex3[0], regexLogs += regex3[1];
        content = content.replace(/<regex( +order *= *\d)?>.*?<\/regex>/gm, '')
            .replace(/\r\n|\r/gm, '\n')
            .replace(/\s*<\|curtail\|>\s*/g, '\n')
            .replace(/\s*<\|join\|>\s*/g, '')
            .replace(/\s*<\|space\|>\s*/g, ' ')
            .replace(/<\|(\\.*?)\|>/g, (match, p1) => { try { return JSON.parse(`"${p1}"`) } catch { return match } });
        return content.replace(/\s*<\|.*?\|>\s*/g, '\n\n')
            .trim().replace(/^.+:/, '\n\n$&')
            .replace(/(?<=\n)\n(?=\n)/g, '');
    };
    let prompt = system || '', regexLogs = '';

    messages.forEach(message => {
        const prefix = '\n\n' + (ROLE_PREFIXS[message.name] || ROLE_PREFIXS[message.role] + (message.name ? `: ${message.name}` : '')) + ': ';
        prompt += `${prefix}${message.content.trim()}`;
    });
    prompt = HyperPmtProcess(prompt);
    if (!claudeMode) prompt += `\n\n${ROLE_PREFIXS['assistant']}:`;
    return {prompt, log: `\n####### Regex:\n${regexLogs}`};
}, ClaudePmtToMsgs = (prompt, oai) => {
    const rounds = prompt.split('\n\n' + ROLE_PREFIXS['user'] + ': ');
    return {
        messages: (oai && rounds.length > 1 ? [{role: 'system', content: rounds[0]}] : [])
            .concat(
                rounds.slice(rounds.length > 1 && 1).flatMap(round => {
                    const turns = round.split(`\n\n${ROLE_PREFIXS['assistant']}:`);
                    return [{role: 'user', content: turns[0].trim()}].concat(turns.slice(1).flatMap(turn => [{role: 'assistant', content: turn.trim()}]));
                }).reduce((acc, current) => {
                    if (acc.length > 0 && (acc[acc.length - 1].role === current.role || !acc[acc.length - 1].content)) {
                        acc[acc.length - 1].content += `\n\n${ROLE_PREFIXS[current.role]}: ` + current.content;
                    } else acc.push(current);
                    return acc;
                }, [])
            ),
        system: !oai && rounds.length > 1 ? rounds[0] : undefined
    };
}, CtoYmsgsConvert = (system, messages) => {
    let YouMessages = [];
    if (messages[messages.length - 1].role !== 'assistant') messages.push({role: 'assistant', content: ''});
    while (messages.length > 1) YouMessages.unshift({answer: messages.pop().content.trim(), question: messages.pop().content.trim()});
    if (system) {
        if (system?.includes(`\n\n${ROLE_PREFIXS['assistant']}:`)) {
            const segments = system.split(`\n\n${ROLE_PREFIXS['assistant']}:`);
            YouMessages.unshift({question: segments[0].trim(), answer: segments.slice(1).join(`\n\n${ROLE_PREFIXS['assistant']}:`).trim()});
        } else YouMessages[0].question = `${system}\n\n${ROLE_PREFIXS['user']}: ${YouMessages[0].question}`.trim();
    }
    return YouMessages;
}, PmtToYouMsgs = prompt => {
    const { system, messages } = ClaudePmtToMsgs(prompt, false);
    return CtoYmsgsConvert(system, messages);
}, youMsgToPmt = (message, withPrefix = true) => {
    return ((message.question.trim() && withPrefix ? `\n\n${ROLE_PREFIXS['user']}: ` : '') + message.question.trim()) +
        (message.answer.trim() ? `\n\n${ROLE_PREFIXS['assistant']}: ${message.answer.trim()}` : '');
}, youPmtProcess = (prompt, ext) => {
    const wedge = { txt: '\u009F', docx: '\u007F' };
    return prompt.split(new RegExp(`\\n\\n(?=${ROLE_PREFIXS['assistant']}:|${ROLE_PREFIXS['user']}:)`, 'g')).join(`\n${wedge[ext]}\n`);
};

((slice, mode) => {
    const messagesClone = JSON.parse(JSON.stringify(slice))
    const {
        prompt, log
    } = HyperProcess("", messagesClone, true)
    console.log(log)

    const youPrompt = prompt.split(/\s*\[-youFileTag-\]\s*/);
    const filePrompt = youPrompt.pop().trim();
    let youMessages = [], youQuery = ""
    if (youPrompt.length > 0) {
        youMessages = PmtToYouMsgs(youPrompt.join('\n\n'));
        youQuery = youMsgToPmt(youMessages.pop(), false);
    }

    const chat = JSON.stringify(youMessages.map(message => ({
        question: youPmtProcess(message.question, mode),
        answer: youPmtProcess(message.answer, mode)
    })))

    return [
        {
            role: "user",
            content: youPmtProcess(filePrompt, mode),
            chat: chat,
            query: youPmtProcess(youQuery, mode),
        }
    ]
})(messages, mode)