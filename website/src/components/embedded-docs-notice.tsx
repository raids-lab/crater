import { Banner } from 'fumadocs-ui/components/banner'
import { getTranslations } from 'next-intl/server'

export async function EmbeddedDocsNotice({ lang }: { lang: string }) {
  if (process.env.CRATER_DOCS_EMBEDDED !== 'true') {
    return null
  }

  const t = await getTranslations({
    locale: lang,
    namespace: 'EmbeddedDocsNotice',
  })

  return (
    <Banner
      height="clamp(3rem, calc(10rem - 8vw), 7rem)"
      className="border-b"
    >
      <p>
        {t('description')}{' '}
        <a
          href={`https://raids-lab.github.io/crater/${lang}/`}
          target="_blank"
          rel="noopener noreferrer"
          className="underline underline-offset-4"
        >
          {t('viewLatest')} ↗
        </a>
      </p>
    </Banner>
  )
}
