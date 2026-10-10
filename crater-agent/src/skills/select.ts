import { platformGuideSkill } from "./platform-guide.js";
import { scopeRefusalSkill } from "./scope-refusal.js";
import type { AgentSkill } from "./types.js";

const overrideAttemptPattern =
  /忽略.{0,20}(指令|规则|提示词)|(?:展示|输出|泄露|告诉我).{0,20}(系统提示词|开发者消息|内部指令)|jailbreak|越狱/iu;
const unrelatedRequestPattern =
  /写诗|作文|小说|编故事|笑话|角色扮演|算命|星座|菜谱|旅游攻略|电影推荐|音乐推荐|股票推荐|政治评论|娱乐新闻/iu;
const platformTopicPattern =
  /crater|平台|集群|作业|任务|jupyter|web\s*ide|批处理|训练|推理|gpu|cpu|内存|显存|资源|镜像|dockerfile|envd|harbor|数据集|模型|文件|挂载|共享|模板|存储|卡时|账户|工单|日志|监控/iu;
const onboardingPattern =
  /新用户|新手|入门|教程|指南|帮助|怎么用|如何使用|能做什么|可以干什么|刚加入|第一次使用|你好|hello|help/iu;
const continuationPattern =
  /^(那|那么|然后|接着|继续|下一步|这个|那个|它|上述|具体|详细|还有|为什么|怎么|如何|怎么办|可以吗|呢)[\s\S]{0,30}$/iu;

export function selectSkill(userMessages: readonly string[]): AgentSkill {
  const latestMessage = userMessages.at(-1)?.trim() ?? "";
  if (
    overrideAttemptPattern.test(latestMessage) ||
    unrelatedRequestPattern.test(latestMessage)
  ) {
    return scopeRefusalSkill;
  }
  if (
    latestMessage === "" ||
    platformTopicPattern.test(latestMessage) ||
    onboardingPattern.test(latestMessage)
  ) {
    return platformGuideSkill;
  }
  const hasPlatformContext = userMessages
    .slice(0, -1)
    .some(
      (message) =>
        platformTopicPattern.test(message) || onboardingPattern.test(message),
    );
  if (hasPlatformContext && continuationPattern.test(latestMessage)) {
    return platformGuideSkill;
  }
  return scopeRefusalSkill;
}
