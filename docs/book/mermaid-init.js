document.addEventListener("DOMContentLoaded", function() {
    const codeBlocks = document.querySelectorAll("pre code.language-mermaid");
    codeBlocks.forEach(function(codeBlock) {
        const preElement = codeBlock.parentElement;
        const diagramText = codeBlock.textContent;
        
        const div = document.createElement("div");
        div.className = "mermaid";
        div.textContent = diagramText;
        
        preElement.parentElement.replaceChild(div, preElement);
    });
    
    // Initialize mermaid with optimized theme settings
    mermaid.initialize({
        startOnLoad: true,
        theme: 'default',
        securityLevel: 'loose',
        flowchart: {
            useMaxWidth: true,
            htmlLabels: true
        }
    });
});
