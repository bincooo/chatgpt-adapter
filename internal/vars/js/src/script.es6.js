/**
 * 该代码为 https://github.com/teralomaniac/clewd 中的片段
 */

const genericFixes = text => text.replace(/(\r\n|\r|\\n)/gm, '\n');

const xmlPlot_merge = (content, mergeTag) => {
    if (/(\n\n|^\s*)xmlPlot:\s*/.test(content)) {
        content = content.replace(/(\n\n|^\s*)(?<!\n\n(Human|Assistant):.*?)xmlPlot:\s*/gs, '$1').replace(/(\n\n|^\s*)xmlPlot: */g, mergeTag.system && mergeTag.human && mergeTag.all ? '\n\nHuman: ' : '$1' );
    }
    mergeTag.all && mergeTag.human && (content = content.replace(/(?:\n\n|^\s*)Human:(.*?(?:\n\nAssistant:|$))/gs, function(match, p1) {return '\n\nHuman:' + p1.replace(/\n\nHuman:\s*/g, '\n\n')}));
    mergeTag.all && mergeTag.assistant && (content = content.replace(/\n\nAssistant:(.*?(?:\n\nHuman:|$))/gs, function(match, p1) {return '\n\nAssistant:' + p1.replace(/\n\nAssistant:\s*/g, '\n\n')}));
    return content;
}, xmlPlot_regex = (content, order) => {
    let matches = content.match(new RegExp(`<regex(?: +order *= *${order})${order === 2 ? '?' : ''}> *"(/?)(.*)\\1(.*?)" *: *"(.*?)" *</regex>`, 'gm'));
    matches && matches.forEach(match => {
        try {
            const reg = /<regex(?: +order *= *\d)?> *"(\/?)(.*)\1(.*?)" *: *"(.*?)" *<\/regex>/.exec(match);
            let reg2 = reg[2], reg3 = reg[3]
            if (reg3.includes('s')) {
                reg2 = reg2.replace(/([^\\])\./g, '$1[\\s\\S]')
                reg3 = reg3.replace('s', 'm')
            }
            content = content.replace(new RegExp(reg2, reg3), JSON.parse(`"${reg[4].replace(/\\?"/g, '\\"')}"`));
        } catch (err) {
            console.log(`[33mRegex error: [0m` + match + '\n' + err);
        }
    });
    return content;
}, xmlPlot = (content) => {
    //一次正则
    content = xmlPlot_regex(content, 1);
    //一次role合并
    const mergeTag = {
        all: !content.includes('<|Merge Disable|>'),
        system: !content.includes('<|Merge System Disable|>'),
        human: !content.includes('<|Merge Human Disable|>'),
        assistant: !content.includes('<|Merge Assistant Disable|>')
    };
    content = xmlPlot_merge(content, mergeTag);
    //自定义插入
    let splitContent = content.split(/\n\n(?=Assistant:|Human:)/g), match;
    while ((match = /<@(\d+)>(.*?)<\/@\1>/gs.exec(content)) !== null) {
        let index = splitContent.length - parseInt(match[1]) - 1;
        index >= 0 && (splitContent[index] += '\n\n' + match[2]);
        content = content.replace(match[0], '');
    }
    content = splitContent.join('\n\n').replace(/<@(\d+)>.*?<\/@\1>/gs, '');
    //二次正则
    content = xmlPlot_regex(content, 2);
    //二次role合并
    content = xmlPlot_merge(content, mergeTag);

    //三次正则
    content = xmlPlot_regex(content, 3);
    //消除空XML tags、两端空白符和多余的\n
    content = content.replace(/<regex( +order *= *\d)?>.*?<\/regex>/gs, '')
        .replace(/\r\n|\r/gm, '\n')
        .replace(/\s*<\|curtail\|>\s*/g, '\n')
        .replace(/\s*<\|join\|>\s*/g, '')
        .replace(/\s*<\|space\|>\s*/g, ' ')
        .replace(/\s*\n\n(H(uman)?|A(ssistant)?): +/g, '\n\n$1: ')
        .replace(/<\|(\\.*?)\|>/g, function(match, p1) {
            try {
                return JSON.parse(`"${p1.replace(/\\?"/g, '\\"')}"`);
            } catch(e) { return match }
        });

    //确保格式正确
    content = content.replace(/(\n\nHuman:(?!.*?\n\nAssistant:).*?|(?<!\n\nAssistant:.*?))$/s, '$&\n\nAssistant:').replace(/\s*<\|noAssistant\|>\s*(.*?)(?:\n\nAssistant:\s*)?$/s, '\n\n$1');
    content.includes('<|reverseHA|>') && (content = content.replace(/\s*<\|reverseHA\|>\s*/g, '\n\n').replace(/Assistant|Human/g, function(match) {return match === 'Human' ? 'Assistant' : 'Human'}).replace(/\n(A|H): /g, function(match, p1) {return p1 === 'A' ? '\nH: ' : '\nA: '}));
    return content.replace(/\s*<\|.*?\|>\s*/g, '\n\n').trim().replace(/^.+:/, '\n\n$&').replace(/(?<=\n)\n(?=\n)/g, '');
};

((messages, Config, Replacements) => {
    try {
        /************************* */
        let { prompt } = ((messages) => {
            const rgxScenario = /^\[Circumstances and context of the dialogue: ([\s\S]+?)\.?\]$/i, rgxPerson = /^\[([\s\S]+?)'s personality: ([\s\S]+?)\]$/i, messagesClone = JSON.parse(JSON.stringify(messages)), realLogs = messagesClone.filter((message => [ 'user', 'assistant' ].includes(message.role))), sampleLogs = messagesClone.filter((message => message.name)), mergedLogs = [ ...sampleLogs, ...realLogs ];
            mergedLogs.forEach(((message, idx) => {
                const next = mergedLogs[idx + 1];
                message.customname = (message => [ 'assistant', 'user' ].includes(message.role) && null != message.name && !(message.name in Replacements))(message);
                if (next) {
                    if ('name' in message && 'name' in next) {
                        if (message.name === next.name) {
                            message.content += '\n' + next.content;
                            next.merged = true;
                        }
                    } else if ('system' !== next.role) {
                        if (next.role === message.role) {
                            message.content += '\n' + next.content;
                            next.merged = true;
                        }
                    } else {
                        message.content += '\n' + next.content;
                        next.merged = true;
                    }
                }
            }));

            const lastAssistant = realLogs.findLast((message => !message.merged && 'assistant' === message.role));
            lastAssistant && Config.Settings.StripAssistant && (lastAssistant.strip = true);
            const lastUser = realLogs.findLast((message => !message.merged && 'user' === message.role));
            lastUser && Config.Settings.StripHuman && (lastUser.strip = true);
            const systemMessages = messagesClone.filter((message => 'system' === message.role && !('name' in message)));
            systemMessages.forEach(((message, idx) => {
                const scenario = message.content.match(rgxScenario)?.[1], personality = message.content.match(rgxPerson);
                if (scenario) {
                    message.content = Config.ScenarioFormat.replace(/{{scenario}}/gim, scenario);
                    message.scenario = true;
                }
                if (3 === personality?.length) {
                    message.content = Config.PersonalityFormat.replace(/{{char}}/gim, personality[1]).replace(/{{personality}}/gim, personality[2]);
                    message.personality = true;
                }
                message.main = 0 === idx;
                message.jailbreak = idx === systemMessages.length - 1;
                ' ' === message.content && (message.discard = true);
            }));

            Config.Settings.AllSamples && !Config.Settings.NoSamples && realLogs.forEach((message => {
                if (![ lastUser, lastAssistant ].includes(message)) {
                    if ('user' === message.role) {
                        message.name = message.customname ? message.name : 'example_user';
                        message.role = 'system';
                    } else if ('assistant' === message.role) {
                        message.name = message.customname ? message.name : 'example_assistant';
                        message.role = 'system';
                    } else if (!message.customname) {
                        throw Error('Invalid role ' + message.name);
                    }
                }
            }));

            Config.Settings.NoSamples && !Config.Settings.AllSamples && sampleLogs.forEach((message => {
                if ('example_user' === message.name) {
                    message.role = 'user';
                } else if ('example_assistant' === message.name) {
                    message.role = 'assistant';
                } else if (!message.customname) {
                    throw Error('Invalid role ' + message.name);
                }
                message.customname || delete message.name;
            }));

            let systems = [];
            const prompt = messagesClone.map(((message, idx) => {
                if (message.merged || message.discard) {
                    return '';
                }
                if (message.content?.length < 1) {
                    return message.content;
                }
                let spacing = '';
                /******************************** */
                if (Config.Settings.xmlPlot) {
                    idx > 0 && (spacing = '\n\n');
                    const prefix = message.customname ? message.role + ': ' + message.name.replaceAll('_', ' ') + ': ' : 'system' !== message.role || message.name ? Replacements[message.name || message.role] + ': ' : 'xmlPlot: ' + Replacements[message.role];
                    return `${spacing}${message.strip ? '' : prefix}${message.content}`;
                } else {
                    /******************************** */
                    idx > 0 && (spacing = systemMessages.includes(message) ? '\n' : '\n\n');
                    const prefix = message.customname ? message.name.replaceAll('_', ' ') + ': ' : 'system' !== message.role || message.name ? Replacements[message.name || message.role] + ': ' : '' + Replacements[message.role];
                    return `${spacing}${message.strip ? '' : prefix}${'system' === message.role ? message.content : message.content.trim()}`;
                } //
            }));

            return {
                prompt: prompt.join(''),
                systems
            };
        })(messages);

        /******************************** */
        const wedge = '\r';
        prompt = Config.Settings.xmlPlot ? xmlPlot(prompt) : `\n\nHuman: ${genericFixes(prompt)}\n\nAssistant:`;
        // prompt = prompt.replace(/\n(?!\nAssistant:\s*$)(?=\n(Human|Assistant):)/gs, '\n' + wedge);
        /******************************** */
        let system;

        let rounds = prompt.replace(/^(?!.*\n\nHuman:)/s, '\n\nHuman:').split('\n\nHuman:');
        messages = rounds.slice(1).flatMap((round, idx) => {
            let turns = round.split('\n\nAssistant:');
            return [{role: 'user', content: turns[0].trim()}].concat(turns.slice(1).flatMap(turn => [{role: 'assistant', content: turn.trim()}]));
        }).reduce((acc, current) => {
            if (acc.length > 0 && (acc[acc.length - 1].role === current.role || !acc[acc.length - 1].content)) {
                acc[acc.length - 1].content += (current.role === 'user' ? 'Human' : 'Assistant').replace(/.*/, '\n\n$&: ') + current.content;
            } else acc.push(current);
            return acc;
        }, []).filter(message => message.content), system = (rounds = rounds[0].split('\n\nAssistant:'))[0].trim();
        rounds.length > 1 && (messages = rounds.slice(1).flatMap(turn => [{role: 'assistant', content: turn.trim()}]).concat(messages));

        if (!Config.Settings.StripAssistant) {
            messages.push({role: "assistant", content: ""})
        }

        if (system) {
            return [ {role: "system", content: system}, ...messages ];
        }
        return messages;

    } catch (err) {
        throw err
    }
})(messages, config, replacements)