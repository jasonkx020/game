declare module 'swagger-ui-dist/swagger-ui-es-bundle.js' {
  interface SwaggerUIOptions {
    domNode?: HTMLElement | null
    url?: string
    spec?: unknown
    docExpansion?: 'list' | 'full' | 'none'
    defaultModelsExpandDepth?: number
    deepLinking?: boolean
    tryItOutEnabled?: boolean
    persistAuthorization?: boolean
    filter?: boolean | string
    syntaxHighlight?: boolean | { activate?: boolean; theme?: string }
  }

  interface SwaggerUIInstance {
    destroy?: () => void
  }

  export default function SwaggerUI(options: SwaggerUIOptions): SwaggerUIInstance
}

declare module 'swagger-ui-dist/swagger-ui.css'
